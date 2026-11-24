package httpx

import (
	"errors"
	"io"
	"net/http"
)

// FormFile reads the multipart part field of r's body, giving its file name and bytes; which may be at most limit bytes in all, or gives the status
// to refuse it with: 413 past the limit, 400 for a body that is not a multipart form with that part.
func FormFile(w http.ResponseWriter, r *http.Request, field string, limit int64) (name string, data []byte, status int) {
	r.Body = http.MaxBytesReader(w, r.Body, limit)
	mr, err := r.MultipartReader()
	if err != nil {
		return "", nil, http.StatusBadRequest
	}
	for {
		p, err := mr.NextPart()
		var tooLarge *http.MaxBytesError
		switch {
		case errors.As(err, &tooLarge):
			return "", nil, http.StatusRequestEntityTooLarge
		case err != nil:
			return "", nil, http.StatusBadRequest
		case p.FormName() != field:
			continue
		}
		data, err = io.ReadAll(p)
		if errors.As(err, &tooLarge) {
			return "", nil, http.StatusRequestEntityTooLarge
		}
		if err != nil {
			return "", nil, http.StatusBadRequest
		}
		return p.FileName(), data, 0
	}
}
