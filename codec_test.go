package schema

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCodec_DecodeRequest(t *testing.T) {
	type queryStruct struct {
		Name string `schema:"name,location=query"`
		Age  int    `schema:"age,location=query"`
	}

	type bodyStruct struct {
		Title   string `schema:"title"`
		Content string `schema:"content"`
	}

	type requestWithBody struct {
		Body bodyStruct `body:"structured"`
	}

	type mixedStruct struct {
		Name string     `schema:"name,location=query"`
		Body bodyStruct `body:"structured"`
	}

	tests := []struct {
		name         string
		method       string
		url          string
		contentType  string
		body         string
		routerParams map[string]string
		result       any
		want         any
	}{
		{
			name:   "query parameters",
			method: "GET",
			url:    "/test?name=John&age=30",
			result: &queryStruct{},
			want:   &queryStruct{Name: "John", Age: 30},
		},
		{
			name:        "JSON body",
			method:      "POST",
			url:         "/test",
			contentType: "application/json",
			body:        `{"title": "Hello", "content": "World"}`,
			result:      &requestWithBody{},
			want:        &requestWithBody{Body: bodyStruct{Title: "Hello", Content: "World"}},
		},
		{
			name:        "mixed query and body",
			method:      "POST",
			url:         "/test?name=John",
			contentType: "application/json",
			body:        `{"title": "Post", "content": "Content"}`,
			result:      &mixedStruct{},
			want:        &mixedStruct{Name: "John", Body: bodyStruct{Title: "Post", Content: "Content"}},
		},
	}

	codec := NewDefaultCodec()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var bodyReader *bytes.Reader
			if tt.body != "" {
				bodyReader = bytes.NewReader([]byte(tt.body))
			} else {
				bodyReader = bytes.NewReader(nil)
			}

			req := httptest.NewRequest(tt.method, tt.url, bodyReader)
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}

			err := codec.DecodeRequest(req, tt.routerParams, tt.result)

			require.NoError(t, err)
			assert.Equal(t, tt.want, tt.result)
		})
	}
}

func TestCodec_DecodeRequest_Integration(t *testing.T) {
	type fullRequest struct {
		// Query parameter
		Filter string `schema:"filter,location=query"`
		// Path parameter
		ID string `schema:"id,location=path"`
		// Header parameter
		APIKey string `schema:"X-Api-Key,location=header"`
		// Body
		Body struct {
			Name  string `schema:"name"`
			Email string `schema:"email"`
		} `body:"structured"`
	}

	codec := NewDefaultCodec()

	body := `{"name": "John Doe", "email": "john@example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/users/123?filter=active", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Api-Key", "secret-key-123")

	routerParams := map[string]string{"id": "123"}

	var result fullRequest
	err := codec.DecodeRequest(req, routerParams, &result)

	require.NoError(t, err)
	assert.Equal(t, "active", result.Filter)
	assert.Equal(t, "123", result.ID)
	assert.Equal(t, "secret-key-123", result.APIKey)
	assert.Equal(t, "John Doe", result.Body.Name)
	assert.Equal(t, "john@example.com", result.Body.Email)
}

func TestCodec_DecodeRequest_Multipart(t *testing.T) {
	type multipartBody struct {
		Name  string `schema:"name"`
		Email string `schema:"email"`
	}

	type multipartRequest struct {
		Body multipartBody `body:"multipart"`
	}

	codec := NewDefaultCodec()

	// Create multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("name", "John Doe")
	_ = writer.WriteField("email", "john@example.com")
	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	var result multipartRequest
	err := codec.DecodeRequest(req, nil, &result)

	require.NoError(t, err)
	assert.Equal(t, "John Doe", result.Body.Name)
	assert.Equal(t, "john@example.com", result.Body.Email)
}

func TestCodec_DecodeRequest_FileUpload(t *testing.T) {
	type fileRequest struct {
		Body []byte `body:"file"`
	}

	codec := NewDefaultCodec()

	fileContent := []byte("file content here")
	req := httptest.NewRequest(http.MethodPost, "/upload", bytes.NewReader(fileContent))
	req.Header.Set("Content-Type", "application/octet-stream")

	var result fileRequest
	err := codec.DecodeRequest(req, nil, &result)

	require.NoError(t, err)
	assert.Equal(t, fileContent, result.Body)
}

func TestCodec_DecodeRequest_FileUploadAsReader(t *testing.T) {
	type fileReaderRequest struct {
		Body io.ReadCloser `body:"file"`
	}

	codec := NewDefaultCodec()

	fileContent := []byte("file content for reader")
	req := httptest.NewRequest(http.MethodPost, "/upload", bytes.NewReader(fileContent))
	req.Header.Set("Content-Type", "application/octet-stream")

	var result fileReaderRequest
	err := codec.DecodeRequest(req, nil, &result)

	require.NoError(t, err)
	require.NotNil(t, result.Body)

	defer result.Body.Close() //nolint:errcheck // test cleanup
	content, err := io.ReadAll(result.Body)
	require.NoError(t, err)
	assert.Equal(t, fileContent, content)
}

func TestCodec_DecodeRequest_MultipartWithFile(t *testing.T) {
	type uploadBody struct {
		Title string        `schema:"title"`
		File  io.ReadCloser `schema:"document"`
	}

	type uploadRequest struct {
		Body uploadBody `body:"multipart"`
	}

	codec := NewDefaultCodec()

	// Create multipart form with text field and file
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("title", "My Document")

	filePart, _ := writer.CreateFormFile("document", "test.txt")
	_, _ = filePart.Write([]byte("document content"))
	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	var result uploadRequest
	err := codec.DecodeRequest(req, nil, &result)

	require.NoError(t, err)
	assert.Equal(t, "My Document", result.Body.Title)
	require.NotNil(t, result.Body.File)

	defer result.Body.File.Close() //nolint:errcheck // test cleanup
	content, err := io.ReadAll(result.Body.File)
	require.NoError(t, err)
	assert.Equal(t, []byte("document content"), content)
}

func TestCodec_DecodeRequest_QueryWithFileUpload(t *testing.T) {
	type mixedRequest struct {
		Version string `schema:"version,location=query"`
		Body    []byte `body:"file"`
	}

	codec := NewDefaultCodec()

	fileContent := []byte("binary data")
	req := httptest.NewRequest(http.MethodPost, "/upload?version=1.0", bytes.NewReader(fileContent))
	req.Header.Set("Content-Type", "application/octet-stream")

	var result mixedRequest
	err := codec.DecodeRequest(req, nil, &result)

	require.NoError(t, err)
	assert.Equal(t, "1.0", result.Version)
	assert.Equal(t, fileContent, result.Body)
}
