package swagger

import (
	"context"
	"html/template"
	"net/http"
	"regexp"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
	swaggerFiles "github.com/swaggo/files"
)

// EchoHandler wraps `http.EchoHandler` into `echo.HandlerFunc`.
func EchoHandler(ctx context.Context, confs ...func(c *Config)) echo.HandlerFunc {
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
		return nil
	}

	// nolint:lll,gofumpt // long string
	var re = regexp.MustCompile(`(.*)(index\.html|doc\.json|favicon-16x16\.png|favicon-32x32\.png|/oauth2-redirect\.html|swagger-ui\.css|swagger-ui\.css\.map|swagger-ui\.js|swagger-ui\.js\.map|swagger-ui-bundle\.js|swagger-ui-bundle\.js\.map|swagger-ui-standalone-preset\.js|swagger-ui-standalone-preset\.js\.map)[\?|.]*`)

	return func(c echo.Context) (err error) {
		var matches []string
		if matches = re.FindStringSubmatch(c.Request().RequestURI); len(matches) != 3 {
			return c.String(http.StatusNotFound, "404 page not found")
		}
		path := matches[2]
		prefix := matches[1]
		handler.Prefix = prefix

		switch path {
		case "index.html":
			tmpConfig := &Config{
				URL:  c.Scheme() + "://" + c.Request().Host + prefix + config.URL,
				Name: config.Name,
			}
			err = index.Execute(c.Response().Writer, tmpConfig)
			if err != nil {
				return
			}
		case "doc.json":
			doc, err1 := ReadDoc(config.Name)
			if err1 != nil {
				return err1
			}
			_, err1 = c.Response().Write([]byte(doc))
			if err1 != nil {
				return err1
			}
		case "":
			err = c.Redirect(http.StatusMovedPermanently, prefix+"index.html")
			if err != nil {
				return
			}
		default:
			handler.ServeHTTP(c.Response().Writer, c.Request())
		}
		return nil
	}
}
