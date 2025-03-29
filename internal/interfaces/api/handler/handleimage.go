package handler

import (
	"encoding/json"
	"expvar"
	"flag"
	"fmt"
	"github.com/mdfriday/hugoverse/internal/domain/content/valueobject"
	"github.com/mdfriday/hugoverse/pkg/hmac"
	"github.com/mdfriday/hugoverse/pkg/images"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
)

// Errors
var (
	// HMAC
	hmacKey = flag.String("hmac-key", "MDFriday service", "hmac key to use for authentication between services")
	hm      = &hmac.HMAC{Key: []byte(*hmacKey)}

	ErrInvalidSize          = fmt.Errorf("invalid size")
	ErrInvalidFileExtension = fmt.Errorf("invalid file extension")
	ErrInvalidBlurAmount    = fmt.Errorf("invalid blur amount")

	minBlurAmount = 1
	maxBlurAmount = 10
	maxImageSize  = 5000 // The max allowed image width/height that can be requested

	imageRequests          = expvar.NewMap("counter_labelmap_dimensions_image_requests_dimension")
	imageRequestsBlur      = expvar.NewInt("image_requests_blur")
	imageRequestsGrayscale = expvar.NewInt("image_requests_grayscale")
)

const defaultBlurAmount = 5

func (s *Handler) ImagesHandler(res http.ResponseWriter, req *http.Request) {
	q := req.URL.Query()
	t := q.Get("type")
	if t != "Image" {
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	s.ApiContentsHandler(res, req)
}

func (s *Handler) ImageHandler(res http.ResponseWriter, req *http.Request) {
	q := req.URL.Query()
	t := q.Get("type")
	if t != "Image" {
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	s.ContentHandler(res, req)
}

func (s *Handler) ImageRandomHandler(res http.ResponseWriter, req *http.Request) {
	// Get the path and query parameters
	params, err := getParams(req)
	if err != nil {
		s.log.Errorf("Error getting params: %v", err)
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	t := "Image"
	status := ""

	post, err := s.contentApp.GetRandomContent(t, status)
	if err != nil {
		s.log.Errorf("Error getting content: %v", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	if post == nil {
		res.WriteHeader(http.StatusNotFound)
		s.log.Printf("Content not found: %s %s", t, status)
		return
	}

	var image valueobject.Image
	if err := json.Unmarshal(post, &image); err != nil {
		s.log.Errorf("Error unmarshalling image: %v", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	absPath, err := s.getUploadAbsPath(image.Asset)
	if err != nil {
		s.log.Errorf("Error getting absolute path: %v", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	width, height, err := images.GetImageDimensions(absPath)
	if err != nil {
		s.log.Errorf("Error getting image dimensions: %v", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}
	image.Width = width
	image.Height = height

	if vErr := s.validateAndRedirect(res, req, params, &image); vErr != nil {
		msgJSON, err := json.Marshal(vErr.Message)
		if err != nil {
			s.log.Errorf("Error marshalling token: %v", err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		j, err := s.res.FmtJSON(msgJSON)
		if err != nil {
			s.log.Errorf("Error formatting JSON: %v", err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		res.WriteHeader(vErr.Code)
		s.res.Json(res, j)
	}
}

func (s *Handler) ImageDummyHandler(res http.ResponseWriter, req *http.Request) {
	// Get the path and query parameters
	p, err := getParams(req)
	if err != nil {
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	fmt.Printf("p: %+v", p)
}

func (s *Handler) validateAndRedirect(w http.ResponseWriter, r *http.Request, p *Params, image *valueobject.Image) *Error {
	if err := validateImageParams(p); err != nil {
		return BadRequest(err.Error())
	}

	width, height := getImageDimensions(p, image)

	w.Header().Set("Cache-Control", "private, no-cache, no-store, must-revalidate")
	w.Header()["Content-Type"] = nil

	path := fmt.Sprintf("/id/%d/%d/%d%s", image.ID, width, height, p.Extension)
	query := url.Values{}

	if p.Blur {
		query.Add("blur", strconv.Itoa(p.BlurAmount))
		imageRequestsBlur.Add(1)
	}

	if p.Grayscale {
		query.Add("grayscale", "")
		imageRequestsGrayscale.Add(1)
	}

	u, err := HMAC(hm, path, query)
	if err != nil {
		return InternalServerError()
	}

	imageRequests.Add(fmt.Sprintf("%0.f",
		math.Max(math.Round(float64(width)/500)*500, math.Round(float64(height)/500)*500)), 1)

	uScheme := r.URL.Scheme
	uHost := r.URL.Host
	uPath := "/image"
	reDir := uScheme + uHost + uPath

	http.Redirect(w, r, fmt.Sprintf("%s%s", reDir, u), http.StatusFound)

	return nil
}

func getImageDimensions(p *Params, databaseImage *valueobject.Image) (width, height int) {
	// Default to the image width/height if 0 is passed
	width = p.Width
	height = p.Height

	if width == 0 {
		width = databaseImage.Width
	}

	if height == 0 {
		height = databaseImage.Height
	}

	return
}

func validateImageParams(p *Params) error {
	if p.Width > maxImageSize {
		return ErrInvalidSize
	}

	if p.Height > maxImageSize {
		return ErrInvalidSize
	}

	if p.Blur && p.BlurAmount < minBlurAmount {
		return ErrInvalidBlurAmount
	}

	if p.Blur && p.BlurAmount > maxBlurAmount {
		return ErrInvalidBlurAmount
	}

	return nil
}

// Params contains all the parameters for a request
type Params struct {
	Width      int
	Height     int
	Blur       bool
	BlurAmount int
	Grayscale  bool
	Extension  string
}

// getParams parses and returns all the path and query parameters
func getParams(r *http.Request) (*Params, error) {
	// Get and validate the width and height from the path parameters
	width, height, err := getSize(r)
	if err != nil {
		return nil, err
	}

	// Get the optional file extension from the path parameters
	extension, err := getFileExtension(r)
	if err != nil {
		return nil, err
	}

	// Get and validate the query parameters for grayscale and blur
	grayscale, blur, blurAmount := getQueryParams(r)

	params := &Params{
		Width:      width,
		Height:     height,
		Blur:       blur,
		BlurAmount: blurAmount,
		Grayscale:  grayscale,
		Extension:  extension,
	}

	return params, nil
}

// getQueryParams returns whether the grayscale and blur queryparams are present
func getQueryParams(r *http.Request) (grayscale bool, blur bool, blurAmount int) {
	if _, ok := r.URL.Query()["grayscale"]; ok {
		grayscale = true
	}

	if _, ok := r.URL.Query()["blur"]; ok {
		blur = true
		blurAmount = defaultBlurAmount

		if val, err := strconv.Atoi(r.URL.Query().Get("blur")); err == nil {
			blurAmount = val
			return
		}
	}

	return
}

// getFileExtension gets the file extension (if present) from the path params, and validates it
func getFileExtension(r *http.Request) (extension string, err error) {
	vars := mux.Vars(r)

	// We only allow the .jpg and .webp extensions, as we only serve jpg and webp images
	// We normalize having no extension since it's an optional path param
	val := strings.ToLower(vars["extension"])

	if val == "" {
		val = ".jpg"
	}

	if val != ".jpg" && val != ".webp" {
		return "", ErrInvalidFileExtension
	}

	return val, nil
}

// getSize gets the image size from the size or the width/height path params, and validates it
func getSize(r *http.Request) (width int, height int, err error) {
	// Check for the size parameter first
	if size, ok := intParam(r, "size"); ok {
		width, height = size, size
	} else {
		// If size doesn't exist, check for width/height
		width, ok = intParam(r, "width")
		if !ok {
			return -1, -1, ErrInvalidSize
		}

		height, ok = intParam(r, "height")
		if !ok {
			return -1, -1, ErrInvalidSize
		}
	}

	return
}

// intParam tries to get a param and convert it to an Integer
func intParam(r *http.Request, name string) (int, bool) {
	vars := mux.Vars(r)

	if val, ok := vars[name]; ok {
		val, err := strconv.Atoi(val)
		return val, err == nil
	}

	return -1, false
}

// HMAC generates and appends an HMAC to a URL path + query params
func HMAC(h *hmac.HMAC, path string, query url.Values) (string, error) {
	hmac, err := h.Create(path + BuildQuery(query))
	if err != nil {
		return "", err
	}

	query.Set("hmac", hmac)
	return path + BuildQuery(query), nil
}
