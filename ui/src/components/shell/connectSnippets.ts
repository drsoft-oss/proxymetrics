export type Language = "curl" | "python" | "node" | "go" | "ruby";

export const LANGUAGES: ReadonlyArray<{ id: Language; label: string }> = [
  { id: "curl",   label: "curl" },
  { id: "python", label: "Python" },
  { id: "node",   label: "Node.js" },
  { id: "go",     label: "Go" },
  { id: "ruby",   label: "Ruby" },
];

const UPSTREAM_EXAMPLE =
  "http://customer-acme-zone-residential-country-us-provider-brightdata-type-residential-price-1200:upstream_password@brd.superproxy.io:22225";

const SECRET_PLACEHOLDER = "<deployment-secret>";

export function makeSnippet(lang: Language, host: string, secret?: string): string {
  const pw = secret && secret.length > 0 ? secret : SECRET_PLACEHOLDER;

  switch (lang) {
    case "curl":
      return `UPSTREAM='${UPSTREAM_EXAMPLE}'
PROXY_USER=$(printf %s "$UPSTREAM" | base64 | tr -d '=' | tr '+/' '-_')
curl --cacert ./proxymetrics-ca.crt \\
  -x "http://$PROXY_USER:${pw}@${host}:8080" \\
  https://example.com`;

    case "python":
      return `import base64, os, requests

UPSTREAM = "${UPSTREAM_EXAMPLE}"
proxy_user = base64.urlsafe_b64encode(UPSTREAM.encode()).decode().rstrip("=")

proxies = {
    "https": f"http://{proxy_user}:${pw}@${host}:8080",
    "http":  f"http://{proxy_user}:${pw}@${host}:8080",
}
r = requests.get("https://example.com", proxies=proxies, verify="./proxymetrics-ca.crt")
print(r.status_code)`;

    case "node":
      return `import { ProxyAgent, fetch } from "undici";
import { readFileSync } from "node:fs";

const UPSTREAM = "${UPSTREAM_EXAMPLE}";
const proxyUser = Buffer.from(UPSTREAM).toString("base64url");

const dispatcher = new ProxyAgent({
  uri: \`http://\${proxyUser}:${pw}@${host}:8080\`,
  requestTls: { ca: readFileSync("./proxymetrics-ca.crt") },
});

const res = await fetch("https://example.com", { dispatcher });
console.log(res.status);`;

    case "go":
      return `package main

import (
    "crypto/tls"
    "crypto/x509"
    "encoding/base64"
    "fmt"
    "io"
    "net/http"
    "net/url"
    "os"
)

func main() {
    upstream := "${UPSTREAM_EXAMPLE}"
    proxyUser := base64.RawURLEncoding.EncodeToString([]byte(upstream))

    proxyURL, _ := url.Parse(fmt.Sprintf("http://%s:${pw}@${host}:8080", proxyUser))

    pool := x509.NewCertPool()
    pem, _ := os.ReadFile("./proxymetrics-ca.crt")
    pool.AppendCertsFromPEM(pem)

    client := &http.Client{Transport: &http.Transport{
        Proxy:           http.ProxyURL(proxyURL),
        TLSClientConfig: &tls.Config{RootCAs: pool},
    }}

    resp, _ := client.Get("https://example.com")
    defer resp.Body.Close()
    body, _ := io.ReadAll(resp.Body)
    fmt.Println(resp.StatusCode, len(body))
}`;

    case "ruby":
      return `require "base64"
require "net/http"
require "openssl"

UPSTREAM = "${UPSTREAM_EXAMPLE}"
proxy_user = Base64.urlsafe_encode64(UPSTREAM, padding: false)

# proxy: ${host}:8080
uri = URI("https://example.com")
http = Net::HTTP.new(uri.host, uri.port,
  "${host}", 8080, proxy_user, "${pw}")
http.use_ssl = true
http.cert_store = OpenSSL::X509::Store.new.tap { |s| s.add_file("./proxymetrics-ca.crt") }
res = http.get(uri.request_uri)
puts res.code`;
  }
}
