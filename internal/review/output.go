package review

import (
	"encoding/json"
	"io"
	"os"
)

// WriteJSON writes the review as JSON to the given writer
func WriteJSON(r *Review, w io.Writer, pretty bool) error {
	encoder := json.NewEncoder(w)
	if pretty {
		encoder.SetIndent("", "  ")
	}
	return encoder.Encode(r)
}

// WriteToFile writes the review as JSON to a file
func WriteToFile(r *Review, path string, pretty bool) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return WriteJSON(r, f, pretty)
}

// ToJSON returns the review as a JSON string
func ToJSON(r *Review, pretty bool) (string, error) {
	var data []byte
	var err error
	if pretty {
		data, err = json.MarshalIndent(r, "", "  ")
	} else {
		data, err = json.Marshal(r)
	}
	if err != nil {
		return "", err
	}
	return string(data), nil
}
