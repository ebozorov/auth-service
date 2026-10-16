window.onload = function () {
  window.ui = SwaggerUIBundle({
    url: "/api/v1/openapi.json",
    dom_id: "#swagger-ui",
    deepLinking: true,
    persistAuthorization: true,
    presets: [SwaggerUIBundle.presets.apis, SwaggerUIStandalonePreset],
    plugins: [SwaggerUIBundle.plugins.DownloadUrl],
    layout: "StandaloneLayout",
  });
};
