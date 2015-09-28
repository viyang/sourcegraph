package httputil

import (
	"compress/gzip"
	"io"
	"net/http"
)

// Internal gzipped writer that satisfies both the (body) writer in gzipped format,
// and maintains the rest of the ResponseWriter interface for header manipulation.
type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
	r        *http.Request // Keep a hold of the Request, for the filter function
	filtered bool          // Has the request been run through the filter function?
	dogzip   bool          // Should we do GZIP compression for this request?
	filterFn func(http.ResponseWriter, *http.Request) bool
}

// Make sure the filter function is applied.
func (w *gzipResponseWriter) applyFilter() {
	if !w.filtered {
		if w.dogzip = w.filterFn(w, w.r); w.dogzip {
			setGzipHeaders(w.Header())
		}
		w.filtered = true
	}
}

// Unambiguous Write() implementation (otherwise both ResponseWriter and Writer
// want to claim this method).
func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	w.applyFilter()
	if w.dogzip {
		// Write compressed
		return w.Writer.Write(b)
	}
	// Write uncompressed
	return w.ResponseWriter.Write(b)
}

// Intercept the WriteHeader call to correctly set the GZIP headers.
func (w *gzipResponseWriter) WriteHeader(code int) {
	w.applyFilter()
	w.ResponseWriter.WriteHeader(code)
}

// Implement WrapWriter interface
func (w *gzipResponseWriter) WrappedWriter() http.ResponseWriter {
	return w.ResponseWriter
}

var (
	defaultFilterTypes = [...]string{
		"text",
		"javascript",
		"json",
	}
)

// Default filter to check if the response should be GZIPped.
// By default, all text (html, css, xml, ...), javascript and json
// content types are candidates for GZIP.
func defaultFilter(w http.ResponseWriter, r *http.Request) bool {
	hdr := w.Header()
	for _, tp := range defaultFilterTypes {
		ok := HeaderMatch(hdr, "Content-Type", HmContains, tp)
		if ok {
			return true
		}
	}
	return false
}

// Gzip compression HTTP handler. If the client supports it, it compresses the response
// written by the wrapped handler. The filter function is called when the response is about
// to be written to determine if compression should be applied. If this argument is nil,
// the default filter will GZIP only content types containing /json|text|javascript/.
func Gzip(h http.Handler, filterFn func(http.ResponseWriter, *http.Request) bool) http.HandlerFunc {
	if filterFn == nil {
		filterFn = defaultFilter
	}
	return func(w http.ResponseWriter, r *http.Request) {
		hdr := w.Header()
		setVaryHeader(hdr)

		// Do nothing on a HEAD request
		if r.Method == "HEAD" {
			h.ServeHTTP(w, r)
			return
		}
		if !acceptsGzip(r.Header) {
			// No gzip support from the client, return uncompressed
			h.ServeHTTP(w, r)
			return
		}

		// Prepare a gzip response container
		gz := gzip.NewWriter(w)
		gzw := &gzipResponseWriter{
			Writer:         gz,
			ResponseWriter: w,
			r:              r,
			filterFn:       filterFn,
		}
		h.ServeHTTP(gzw, r)
		// Iff the handler completed successfully (no panic) and GZIP was indeed used, close the gzip writer,
		// which seems to generate a Write to the underlying writer.
		if gzw.dogzip {
			gz.Close()
		}
	}
}

// Add the vary by "accept-encoding" header if it is not already set.
func setVaryHeader(hdr http.Header) {
	if !HeaderMatch(hdr, "Vary", HmContains, "accept-encoding") {
		hdr.Add("Vary", "Accept-Encoding")
	}
}

// Checks if the client accepts GZIP-encoded responses.
func acceptsGzip(hdr http.Header) bool {
	ok := HeaderMatch(hdr, "Accept-Encoding", HmContains, "gzip")
	if !ok {
		ok = HeaderMatch(hdr, "Accept-Encoding", HmEquals, "*")
	}
	return ok
}

func setGzipHeaders(hdr http.Header) {
	// The content-type will be explicitly set somewhere down the path of handlers
	hdr.Set("Content-Encoding", "gzip")
	hdr.Del("Content-Length")
}
