package devin

// loginSuccessHTML is served to the browser after a successful OAuth callback.
// Ported from upstream CLIProxyAPI internal/auth/devin/devin_auth.go (f94752762bb9).
const loginSuccessHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>Authentication Successful - Devin</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; display: flex; justify-content: center; align-items: center; height: 100vh; margin: 0; background: #0f172a; color: #f8fafc; }
        .card { background: #1e293b; padding: 2.5rem; border-radius: 12px; box-shadow: 0 8px 30px rgba(0,0,0,0.4); text-align: center; max-width: 420px; }
        h2 { margin-top: 0; color: #38bdf8; }
        p { color: #94a3b8; font-size: 15px; }
    </style>
</head>
<body>
    <div class="card">
        <h2>Authentication Complete</h2>
        <p>You have successfully logged in to Devin via LLMHub.</p>
        <p>You may safely close this window and return to your terminal.</p>
    </div>
</body>
</html>`

// loginFailureHTML is served to the browser when the OAuth callback carries an
// error or is missing the authorization code.
const loginFailureHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>Authentication Failed - Devin</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; display: flex; justify-content: center; align-items: center; height: 100vh; margin: 0; background: #0f172a; color: #f8fafc; }
        .card { background: #1e293b; padding: 2.5rem; border-radius: 12px; box-shadow: 0 8px 30px rgba(0,0,0,0.4); text-align: center; max-width: 420px; }
        h2 { margin-top: 0; color: #f87171; }
        p { color: #94a3b8; font-size: 15px; }
    </style>
</head>
<body>
    <div class="card">
        <h2>Authentication Failed</h2>
        <p>Devin authentication encountered an error: %s</p>
        <p>Please check your terminal and try again.</p>
    </div>
</body>
</html>`
