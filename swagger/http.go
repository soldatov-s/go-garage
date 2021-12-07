package swagger

import (
	"context"
	"html/template"
	"net/http"
	"regexp"

	"github.com/rs/zerolog"
	swaggerFiles "github.com/swaggo/files"
)

// Handler wraps `http.Handler` into `http.HandlerFunc`.
func Handler(ctx context.Context, confs ...func(c *Config)) http.HandlerFunc {
	logger := zerolog.Ctx(ctx)
	handler := swaggerFiles.Handler

	config := &Config{
		URL: "doc.json",
	}

	for _, c := range confs {
		c(config)
	}

	// create a template with name
	t := template.New("swagger_index.html")
	index, err := t.Parse(IndexTempl)
	if err != nil {
		logger.Err(err).Msg("parse template")
	}

	// nolint:lll,gofumpt // long string
	var re = regexp.MustCompile(`(.*)(index\.html|doc\.json|favicon-16x16\.png|favicon-32x32\.png|/oauth2-redirect\.html|swagger-ui\.css|swagger-ui\.css\.map|swagger-ui\.js|swagger-ui\.js\.map|swagger-ui-bundle\.js|swagger-ui-bundle\.js\.map|swagger-ui-standalone-preset\.js|swagger-ui-standalone-preset\.js\.map)[\?|.]*`)

	return func(w http.ResponseWriter, r *http.Request) {
		var matches []string
		if matches = re.FindStringSubmatch(r.RequestURI); len(matches) != 3 {
			w.WriteHeader(http.StatusNotFound)
			if _, err := w.Write([]byte("404 page not found")); err != nil {
				logger.Err(err).Msg("Error write bytes")
			}
			return
		}
		path := matches[2]
		prefix := matches[1]
		handler.Prefix = prefix

		switch path {
		case "index.html":
			tmpConfig := &Config{
				URL:  r.URL.Scheme + "://" + r.Host + prefix + config.URL,
				Name: config.Name,
			}
			err := index.Execute(w, tmpConfig)
			if err != nil {
				logger.Err(err).Msg("Error build template")
			}
		case "doc.json":
			doc, err := ReadDoc(config.Name)
			if err != nil {
				logger.Err(err).Msg("Error read doc")
			}
			_, err = w.Write([]byte(doc))
			if err != nil {
				logger.Err(err).Msg("Error write bytes")
			}
		case "":
			http.Redirect(w, r, prefix+"index.html", http.StatusMovedPermanently)
		default:
			handler.ServeHTTP(w, r)
		}
	}
}
