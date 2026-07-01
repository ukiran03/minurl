package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"path"
	"strings"
)

type envelope map[string]any

func (app *application) writeJSON(
	w http.ResponseWriter, status int, data envelope, headers http.Header,
) error {
	js, err := json.Marshal(data)
	if err != nil {
		return err
	}
	js = append(js, '\n')

	for key, values := range headers {
		for _, v := range values {
			w.Header().Add(key, v)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(js)

	return nil
}

func (app *application) readJSON(
	w http.ResponseWriter, r *http.Request, dst any,
) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1_048_576)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	err := dec.Decode(dst)
	if err != nil {
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError
		var invalidUnmarshalError *json.InvalidUnmarshalError
		var maxBytesError *http.MaxBytesError

		switch {
		case errors.As(err, &syntaxError):
			return fmt.Errorf(
				"body contains badly-formed JSON (at character %d)",
				syntaxError.Offset)

		case errors.Is(err, io.ErrUnexpectedEOF):
			return errors.New("body contains badly-formed JSON")

		case errors.As(err, &unmarshalTypeError):
			if unmarshalTypeError.Field != "" {
				return fmt.Errorf(
					"body contains incorrect JSON type for field %q",
					unmarshalTypeError.Field)
			}
			return fmt.Errorf(
				"body contains incorrect JSON type (at character %d)",
				unmarshalTypeError.Offset)

		case errors.Is(err, io.EOF):
			return errors.New("body must not be empty")

		case strings.HasPrefix(err.Error(), "json: unknown field "):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field")
			return fmt.Errorf("body contains unknown key %s", fieldName)

		case errors.As(err, &maxBytesError):
			return fmt.Errorf(
				"body must not be larger than %d bytes", maxBytesError.Limit)

		case errors.As(err, &invalidUnmarshalError):
			panic(err)

		default:
			return err
		}
	}
	err = dec.Decode(&struct{}{})
	if !errors.Is(err, io.EOF) {
		return errors.New("body must only contain a single JSON value")
	}

	return nil
}

// -- Validate URL & Sanitize --

func processURL(inputRawURL string) (string, error) {
	// DoS vector prevention
	cleanInput := strings.TrimSpace(inputRawURL)
	if len(cleanInput) > 2048 {
		return "", errors.New("URL too long")
	}
	if cleanInput == "" {
		return "", errors.New("empty URL")
	}

	// ensure scheme
	cleanInput = ensureScheme(cleanInput)

	parsedURL, err := url.ParseRequestURI(cleanInput)
	if err != nil {
		return "", ErrInvalidURL
	}

	// enforce HTTP/HTTPS only
	scheme := strings.ToLower(parsedURL.Scheme)
	if scheme != "http" && scheme != "https" {
		return "", ErrInvalidScheme
	}

	host := parsedURL.Hostname()
	if host == "" {
		return "", ErrMissingHost
	}

	// SSRF protection (block internal IPs/localhost)
	if isLocalOrInvalidHost(host) {
		return "", ErrUnsafeHost
	}

	return sanitizeURL(parsedURL), nil
}

func ensureScheme(raw string) string {
	lower := strings.ToLower(raw)
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		return raw
	}

	if strings.HasPrefix(raw, "://") {
		return "https" + raw
	}

	// Default to secure https
	return "https://" + raw
}

func sanitizeURL(u *url.URL) string {
	u.Scheme = strings.ToLower(u.Scheme)
	u.Host = strings.ToLower(u.Host)

	// strip default ports to deduplicate DB records
	if (u.Scheme == "http" && u.Port() == "80") ||
		(u.Scheme == "https" && u.Port() == "443") {
		u.Host = u.Hostname()
	}

	// clean path redundancies (e.g., /foo/bar/../baz -> /foo/baz)
	if u.Path != "" {
		u.Path = path.Clean(u.Path)
	} else {
		u.Path = "/"
	}

	return u.String()
}

func isLocalOrInvalidHost(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	if ip != nil {
		return ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified()
	}
	return false
}
