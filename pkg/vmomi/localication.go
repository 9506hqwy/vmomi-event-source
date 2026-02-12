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

var catalogCache = make(map[string]string, 0)

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

	return &catalog, nil
}

func getCatalogValueFromCache(key string) *string {
	value, ok := catalogCache[key]
	if ok && value != "" {
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

func parseCatalog(catalog *string) map[string]string {
	lines := strings.Split(*catalog, "\n")

	values := make(map[string]string, len(lines))
	for _, line := range lines {
		kv := strings.SplitN(line, "=", kvMin+1)
		if len(kv) > kvMin {
			key := strings.TrimSpace(kv[0])
			value := strings.TrimSpace(kv[1])
			values[key] = strings.Trim(value, "\"")
		}
	}

	return values
}

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
