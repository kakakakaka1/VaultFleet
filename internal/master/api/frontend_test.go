package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFrontendRootServesAcceptanceConsole(t *testing.T) {
	router := newFrontendPlaceholderTestRouter()

	w := getFrontendPlaceholder(t, router, "/")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "text/html")
	assert.Contains(t, w.Body.String(), "VaultFleet")
	assert.Contains(t, w.Body.String(), `name="username"`)
	assert.Contains(t, w.Body.String(), `name="password"`)
	assert.Contains(t, w.Body.String(), "Nodes")
	assert.Contains(t, w.Body.String(), "Backup Now")
	assert.Contains(t, w.Body.String(), "Browse Files")
}

func TestFrontendDoesNotPrefetchProtectedAPIsBeforeAuth(t *testing.T) {
	router := newFrontendPlaceholderTestRouter()

	w := getFrontendPlaceholder(t, router, "/")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `if (!state.initialized) {`)
	assert.Contains(t, w.Body.String(), `if (!auth.authenticated) { showAuth(); return; }`)
	assert.Contains(t, w.Body.String(), `$("auth-mode").textContent = "Login";`)
}

func TestFrontendAcceptancePaths(t *testing.T) {
	router := newFrontendPlaceholderTestRouter()

	for _, path := range []string{"/", "/dashboard", "/nodes", "/nodes/agent-1", "/settings"} {
		t.Run(path, func(t *testing.T) {
			w := getFrontendPlaceholder(t, router, path)

			require.Equal(t, http.StatusOK, w.Code)
			assert.Contains(t, w.Body.String(), "VaultFleet")
			assert.Contains(t, w.Body.String(), "app-root")
		})
	}
}

func TestFrontendLoadsNodeDetailFromAcceptancePath(t *testing.T) {
	router := newFrontendPlaceholderTestRouter()

	w := getFrontendPlaceholder(t, router, "/nodes/agent-1")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "function routeAgentID()")
	assert.Contains(t, w.Body.String(), `window.location.pathname`)
	assert.Contains(t, w.Body.String(), `state.currentAgentId = routeAgentID()`)
	assert.Contains(t, w.Body.String(), `history.pushState`)
}

func TestFrontendRendersClickableFileBrowserEntries(t *testing.T) {
	router := newFrontendPlaceholderTestRouter()

	w := getFrontendPlaceholder(t, router, "/nodes/agent-1")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "function renderBrowse")
	assert.Contains(t, w.Body.String(), `data-browse-path`)
	assert.Contains(t, w.Body.String(), `entry.type === "dir"`)
	assert.Contains(t, w.Body.String(), `document.querySelectorAll("[data-browse-path]")`)
}

func TestFrontendPlaceholderDoesNotServeBackendRoutes(t *testing.T) {
	router := newFrontendPlaceholderTestRouter()

	for _, path := range []string{"/api/missing", "/ws/missing"} {
		t.Run(path, func(t *testing.T) {
			w := getFrontendPlaceholder(t, router, path)

			require.Equal(t, http.StatusNotFound, w.Code)
			assert.NotContains(t, w.Header().Get("Content-Type"), "text/html")
			assert.NotContains(t, w.Body.String(), "VaultFleet")
		})
	}
}

func newFrontendPlaceholderTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	RegisterFrontendRoutes(router)
	return router
}

func getFrontendPlaceholder(t *testing.T, router http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, path, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}
