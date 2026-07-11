package protocol

import "strings"

type UploadAction struct {
	URL       string `json:"url"`
	Method    string `json:"method"`
	FieldName string `json:"fieldName"`
	Multiple  bool   `json:"multiple"`
	MaxSize   *int   `json:"maxSize"`
	Accept    string `json:"accept"`
}

type UploadFile struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	Size      int    `json:"size"`
	ContentID string `json:"contentId"`
}

type UploadResponse struct {
	URL string `json:"url"`
	ID  string `json:"id"`
}

type UploadTransportResult struct {
	Type     string          `json:"type"`
	Status   int             `json:"status"`
	Response *UploadResponse `json:"response"`
}

type UploadExecutionInput struct {
	Action  UploadAction            `json:"action"`
	Files   []UploadFile            `json:"files"`
	Results []UploadTransportResult `json:"results"`
}

type UploadPart struct {
	Name      string `json:"name"`
	FileName  string `json:"fileName"`
	ContentID string `json:"contentId"`
}

type UploadRequest struct {
	Method string     `json:"method"`
	URL    string     `json:"url"`
	Part   UploadPart `json:"part"`
}

type UploadExecutionResult struct {
	OK         bool            `json:"ok"`
	Code       string          `json:"code,omitempty"`
	FileIndex  int             `json:"fileIndex,omitempty"`
	Requests   []UploadRequest `json:"requests"`
	FieldValue any             `json:"fieldValue"`
}

func ExecuteUpload(input UploadExecutionInput) UploadExecutionResult {
	action := input.Action
	if !action.Multiple && len(input.Files) > 1 {
		return uploadFailure("MULTIPLE_FILES_NOT_ALLOWED", 1, nil)
	}
	for index, file := range input.Files {
		if action.MaxSize != nil && file.Size > *action.MaxSize {
			return uploadFailure("FILE_TOO_LARGE", index, nil)
		}
		if !uploadMatchesAccept(file, action.Accept) {
			return uploadFailure("UNSUPPORTED_FILE_TYPE", index, nil)
		}
	}

	requests := make([]UploadRequest, 0, len(input.Files))
	values := make([]any, 0, len(input.Files))
	for index, file := range input.Files {
		requests = append(requests, uploadRequestFor(action, file))
		if index >= len(input.Results) || input.Results[index].Type != "success" {
			return uploadFailure("UPLOAD_REQUEST_FAILED", index, requests)
		}
		value, ok := uploadResponseValue(input.Results[index].Response)
		if !ok {
			return uploadFailure("INVALID_UPLOAD_RESPONSE", index, requests)
		}
		values = append(values, value)
	}

	var fieldValue any
	if action.Multiple {
		fieldValue = values
	} else if len(values) > 0 {
		fieldValue = values[0]
	}
	return UploadExecutionResult{OK: true, Requests: requests, FieldValue: fieldValue}
}

func uploadFailure(code string, fileIndex int, requests []UploadRequest) UploadExecutionResult {
	if requests == nil {
		requests = []UploadRequest{}
	}
	return UploadExecutionResult{OK: false, Code: code, FileIndex: fileIndex, Requests: requests, FieldValue: nil}
}

func uploadMatchesAccept(file UploadFile, accept string) bool {
	if accept == "" {
		return true
	}
	fileName := strings.ToLower(file.Name)
	mime := strings.ToLower(file.Type)
	for _, rawToken := range strings.Split(accept, ",") {
		token := strings.ToLower(strings.TrimSpace(rawToken))
		switch {
		case token == "":
			continue
		case strings.HasPrefix(token, ".") && strings.HasSuffix(fileName, token):
			return true
		case strings.HasSuffix(token, "/*") && strings.HasPrefix(mime, strings.TrimSuffix(token, "*")):
			return true
		case mime == token:
			return true
		}
	}
	return false
}

func uploadRequestFor(action UploadAction, file UploadFile) UploadRequest {
	method := action.Method
	if method == "" {
		method = "POST"
	}
	fieldName := action.FieldName
	if fieldName == "" {
		fieldName = "file"
	}
	return UploadRequest{
		Method: method,
		URL:    action.URL,
		Part:   UploadPart{Name: fieldName, FileName: file.Name, ContentID: file.ContentID},
	}
}

func uploadResponseValue(response *UploadResponse) (string, bool) {
	if response == nil {
		return "", false
	}
	if response.URL != "" {
		return response.URL, true
	}
	if response.ID != "" {
		return response.ID, true
	}
	return "", false
}
