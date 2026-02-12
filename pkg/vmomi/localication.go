package vmomi

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/mo"

	"github.com/9506hqwy/vmomi-event-source/pkg/flag"
	sx "github.com/9506hqwy/vmomi-event-source/pkg/vmomi/sessionex"
)

const kvMin = int(1)
const LineContinue = "\\"

var catalogCache = make(map[string]string, 0)
var catalogTextCache = make(map[string]string, 0)

func GetLocalizationManager(
	ctx context.Context,
	c *vim25.Client,
) (*mo.LocalizationManager, error) {
	if c.ServiceContent.LocalizationManager == nil {
		return nil, nil
	}

	pc := property.DefaultCollector(c)

	var l mo.LocalizationManager
	_, err := sx.ExecCallAPI(
		ctx,
		func(cctx context.Context) (int, error) {
			return 0, pc.RetrieveOne(
				cctx,
				*c.ServiceContent.LocalizationManager,
				[]string{"catalog"},
				&l,
			)
		},
	)
	if err != nil {
		return nil, err
	}

	return &l, nil
}

func GetLocalizationCatalogValue(
	ctx context.Context,
	c *vim25.Client,
	lm *mo.LocalizationManager,
	locale string,
	moduleName string,
	key string,
) (*string, error) {
	value := getCatalogValueFromCache(key)
	if value != nil {
		return value, nil
	}

	url := getLocalizationCatalogURI(lm, locale, moduleName)
	if url == nil {
		return nil, nil
	}

	catalogText := getCatalogTextFromCache(*url)
	if catalogText != nil {
		// Return if not found key in parsed content.
		return nil, nil
	}

	catalogText, err := getLocalizationCatalog(ctx, c, *url)
	if err != nil {
		return nil, err
	}

	updateCatalogCache(catalogText)

	value = getCatalogValueFromCache(key)
	return value, nil
}

func getLocalizationCatalogURI(
	lm *mo.LocalizationManager,
	locale string,
	moduleName string,
) *string {
	uri := findLocalizationCatalogURI(lm, locale, moduleName)
	if uri != nil {
		return uri
	}

	//revive:disable:add-constant
	locales := strings.SplitN(locale, "_", 2)
	//revive:enable:add-constant
	return findLocalizationCatalogURI(lm, locales[0], moduleName)
}

func findLocalizationCatalogURI(
	lm *mo.LocalizationManager,
	locale string,
	moduleName string,
) *string {
	for _, catalog := range lm.Catalog {
		if catalog.Locale == locale && catalog.ModuleName == moduleName {
			return &catalog.CatalogUri
		}
	}

	return nil
}

func getLocalizationCatalog(
	ctx context.Context,
	c *vim25.Client,
	uri string,
) (*string, error) {
	cached := getCatalogTextFromCache(uri)
	if cached != nil {
		return cached, nil
	}

	noVerifySSL, ok := ctx.Value(flag.TargetNoVerifySSLKey{}).(bool)
	if !ok {
		return nil, errors.New("target_no_verify_ssl not found in context")
	}

	req, err := createRequest(ctx, c, uri)
	if err != nil {
		return nil, err
	}

	tlsConfig := tls.Config{InsecureSkipVerify: noVerifySSL}
	res, err := send(req, &tlsConfig)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	catalogBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	catalog := string(catalogBytes)
	updateCatalogTextCache(uri, &catalog)

	return &catalog, nil
}

func getCatalogValueFromCache(key string) *string {
	value, ok := catalogCache[key]
	if ok && len(value) != Empty {
		return &value
	}

	return nil
}

func updateCatalogCache(catalogText *string) {
	catalog := parseCatalog(catalogText)
	for k, v := range catalog {
		catalogCache[k] = v
	}
}

func getCatalogTextFromCache(key string) *string {
	value, ok := catalogTextCache[key]
	if ok && len(value) != Empty {
		return &value
	}

	return nil
}

func updateCatalogTextCache(key string, catalogText *string) {
	catalogTextCache[key] = *catalogText
}

//revive:disable:cognitive-complexity

func parseCatalog(catalog *string) map[string]string {
	lines := strings.Split(*catalog, "\n")

	values := make(map[string]string, len(lines))
	//revive:disable:add-constant
	for i := 0; i < len(lines); i++ {
		//revive:enable:add-constant
		line := lines[i]
		if strings.HasPrefix(line, "#") {
			continue
		}

		kv := strings.SplitN(line, "=", kvMin+1)
		if len(kv) <= kvMin {
			continue
		}

		key := strings.TrimSpace(kv[0])
		value := strings.TrimSpace(kv[1])

		var valueBuilder strings.Builder
		_, err := valueBuilder.WriteString(strings.TrimSuffix(value, LineContinue))
		if err != nil {
			panic(err)
		}

		for strings.HasSuffix(value, "\\") {
			// Continue if line ends with backslash.
			i++
			value = lines[i]
			_, err = valueBuilder.WriteString(strings.TrimSuffix(value, LineContinue))
			if err != nil {
				panic(err)
			}
		}

		values[key] = strings.Trim(valueBuilder.String(), "\"")
	}

	return values
}

//revive:enable:cognitive-complexity

func createRequest(
	ctx context.Context,
	c *vim25.Client,
	uri string,
) (*http.Request, error) {
	url := c.URL()
	url.Path = uri

	req, err := http.NewRequestWithContext(ctx, "GET", url.String(), nil)
	if err != nil {
		return nil, err
	}

	return req, nil
}

func send(req *http.Request, tlsConfig *tls.Config) (*http.Response, error) {
	client := http.Client{}

	client.Transport = &http.Transport{
		TLSClientConfig: tlsConfig,
	}

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	return res, nil
}
