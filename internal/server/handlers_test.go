package server

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DariSorokina/yp-gophermart.git/internal/app"
	"github.com/DariSorokina/yp-gophermart.git/internal/client"
	"github.com/DariSorokina/yp-gophermart.git/internal/config"
	"github.com/DariSorokina/yp-gophermart.git/internal/cookie"
	"github.com/DariSorokina/yp-gophermart.git/internal/database"
	"github.com/DariSorokina/yp-gophermart.git/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testRequest(t *testing.T, ts *httptest.Server, method, path string, clientID int, requestBody io.Reader) (*http.Response, string) {
	req, err := http.NewRequest(method, ts.URL+path, requestBody)
	require.NoError(t, err)

	client := &http.Client{
		Transport: &http.Transport{
			DisableCompression: true,
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	if clientID != 0 {
		clientIDcookie := cookie.CreateCookieClientID(clientID)
		req.AddCookie(clientIDcookie)
	}

	result, err := client.Do(req)
	require.NoError(t, err)
	defer result.Body.Close()

	resultBody, err := io.ReadAll(result.Body)
	require.NoError(t, err)

	return result, string(resultBody)
}

func TestRouter(t *testing.T) {
	flagConfig := config.ParseFlags()

	var l *logger.Logger
	var err error
	if l, err = logger.CreateLogger(flagConfig.FlagLogLevel); err != nil {
		log.Fatal("Failed to create logger:", err)
	}

	storage, err := database.NewPostgresqlDB(flagConfig.FlagDatabaseURI, l)
	if err != nil {
		panic(err)
	}
	defer storage.Close()

	app := app.NewApp(storage, l)
	accuralSystem := client.NewAccrualSystem(flagConfig.FlagAccrualSystemAddress, app, l)
	go client.Run(accuralSystem)

	serv := NewServer(app, flagConfig, l)
	testServer := httptest.NewServer(serv.newRouter())
	defer testServer.Close()

	type expectedData struct {
		expectedContentType string
		expectedStatusCode  int
		expectedBody        string
	}

	testCases := []struct {
		name         string
		method       string
		clientID     int
		requestBody  io.Reader
		requestPath  string
		expectedData expectedData
	}{
		{
			name:        "handler: registerHandler, test: StatusOK",
			method:      http.MethodPost,
			clientID:    0,
			requestBody: bytes.NewBuffer([]byte("{\"login\": \"ephemeral\",\"password\": \"qwerty\"}")),
			requestPath: "/api/user/register",
			expectedData: expectedData{
				expectedContentType: "",
				expectedStatusCode:  http.StatusOK,
				expectedBody:        "",
			},
		},
		{
			name:        "handler: registerHandler, test: StatusBadRequest",
			method:      http.MethodPost,
			clientID:    0,
			requestBody: bytes.NewBuffer([]byte("{\"login\": \"ephemeral\"}")),
			requestPath: "/api/user/register",
			expectedData: expectedData{
				expectedContentType: "text/plain; charset=utf-8",
				expectedStatusCode:  http.StatusBadRequest,
				expectedBody:        "Missing login or password\n",
			},
		},
		{
			name:        "handler: registerHandler, test: StatusConflict",
			method:      http.MethodPost,
			clientID:    0,
			requestBody: bytes.NewBuffer([]byte("{\"login\": \"ephemeral\",\"password\": \"qwerty\"}")),
			requestPath: "/api/user/register",
			expectedData: expectedData{
				expectedContentType: "",
				expectedStatusCode:  http.StatusConflict,
				expectedBody:        "",
			},
		},
		{
			name:        "handler: loginHandler, test: StatusOK",
			method:      http.MethodPost,
			clientID:    0,
			requestBody: bytes.NewBuffer([]byte("{\"login\": \"ephemeral\",\"password\": \"qwerty\"}")),
			requestPath: "/api/user/login",
			expectedData: expectedData{
				expectedContentType: "",
				expectedStatusCode:  http.StatusOK,
				expectedBody:        "",
			},
		},
		{
			name:        "handler: loginHandler, test: StatusBadRequest",
			method:      http.MethodPost,
			clientID:    0,
			requestBody: bytes.NewBuffer([]byte("{\"login\": \"ephemeral\"}")),
			requestPath: "/api/user/login",
			expectedData: expectedData{
				expectedContentType: "text/plain; charset=utf-8",
				expectedStatusCode:  http.StatusBadRequest,
				expectedBody:        "Missing login or password\n",
			},
		},
		{
			name:        "handler: loginHandler, test: StatusUnauthorized",
			method:      http.MethodPost,
			clientID:    0,
			requestBody: bytes.NewBuffer([]byte("{\"login\": \"ephemeral\",\"password\": \"qwerty1\"}")),
			requestPath: "/api/user/login",
			expectedData: expectedData{
				expectedContentType: "",
				expectedStatusCode:  http.StatusUnauthorized,
				expectedBody:        "",
			},
		},
		{
			name:        "handler: postOrderNumberHandler, test: StatusAccepted",
			method:      http.MethodPost,
			clientID:    1,
			requestBody: bytes.NewBuffer([]byte("12345678903")),
			requestPath: "/api/user/orders",
			expectedData: expectedData{
				expectedContentType: "",
				expectedStatusCode:  http.StatusAccepted,
				expectedBody:        "",
			},
		},
		{
			name:        "handler: postOrderNumberHandler, test: StatusOK",
			method:      http.MethodPost,
			clientID:    1,
			requestBody: bytes.NewBuffer([]byte("12345678903")),
			requestPath: "/api/user/orders",
			expectedData: expectedData{
				expectedContentType: "",
				expectedStatusCode:  http.StatusOK,
				expectedBody:        "",
			},
		},
		{
			name:        "handler: postOrderNumberHandler, test: StatusUnprocessableEntity",
			method:      http.MethodPost,
			clientID:    1,
			requestBody: bytes.NewBuffer([]byte("12345678900")),
			requestPath: "/api/user/orders",
			expectedData: expectedData{
				expectedContentType: "",
				expectedStatusCode:  http.StatusUnprocessableEntity,
				expectedBody:        "",
			},
		},
	}
	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result, resultBody := testRequest(t, testServer, test.method, test.requestPath, test.clientID, test.requestBody)
			defer result.Body.Close()
			assert.Equal(t, test.expectedData.expectedStatusCode, result.StatusCode)
			assert.Equal(t, test.expectedData.expectedContentType, result.Header.Get("Content-Type"))
			assert.Equal(t, test.expectedData.expectedBody, string(resultBody))
		})
	}
}
