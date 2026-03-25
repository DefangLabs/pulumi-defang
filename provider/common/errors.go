// Ported from https://github.com/DefangLabs/defang-mvp/blob/main/pulumi/shared/error.ts
package common

func GetErrorHtml(title string, h1 string, message string) string {
	return `<!DOCTYPE html>
<html>
<head>
    <title>Defang | ` + title + `</title>
    <style>
    body {
        font-family: 'Exo', sans-serif;
        background-color: #1d1d1d;
        background-image: radial-gradient(circle at 50% 0, rgba(239, 152, 207, .2), rgba(0, 0, 0, 0) 57%),
          radial-gradient(circle at 0 20%, rgba(122, 167, 255, .25), rgba(0, 0, 0, 0) 42%);
        color: #f1f3fa;
        display: flex;
        justify-content: center;
        align-items: center;
        height: 100vh;
        margin: 0;
    }
    .container {
        text-align: center;
    }
    .message {
        font-size: 1.5em;
    }
    </style>
</head>
<body>
    <div class="container">
    <h1>` + h1 + `</h1>
    <p class="message">` + message + `</p>
    </div>
</body>
</html>`
}
