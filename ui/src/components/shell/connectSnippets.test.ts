import { describe, it, expect } from "vitest";
import { LANGUAGES, makeSnippet, type Language } from "./connectSnippets";

describe("connectSnippets", () => {
  it("exposes exactly five languages in display order", () => {
    expect(LANGUAGES.map((l) => l.id)).toEqual(["curl", "python", "node", "go", "ruby"]);
    expect(LANGUAGES.map((l) => l.label)).toEqual(["curl", "Python", "Node.js", "Go", "Ruby"]);
  });

  it("substitutes the host into every snippet", () => {
    for (const { id } of LANGUAGES) {
      const out = makeSnippet(id as Language, "proxy.example.com");
      expect(out).toContain("proxy.example.com:8080");
    }
  });

  it("falls back to <deployment-secret> placeholder when no secret is provided", () => {
    for (const { id } of LANGUAGES) {
      expect(makeSnippet(id as Language, "host")).toContain("<deployment-secret>");
    }
  });

  it("inlines the deployment secret when provided and omits the placeholder", () => {
    for (const { id } of LANGUAGES) {
      const out = makeSnippet(id as Language, "host", "s3cr3t");
      expect(out).toContain("s3cr3t");
      expect(out).not.toContain("<deployment-secret>");
    }
  });

  it("curl snippet uses --cacert with the local cert path", () => {
    expect(makeSnippet("curl", "host")).toMatch(/--cacert\s+\.\/proxymetrics-ca\.crt/);
  });

  it("python snippet imports requests and uses verify=", () => {
    const py = makeSnippet("python", "host");
    expect(py).toContain("import base64, os, requests");
    expect(py).toContain('verify="./proxymetrics-ca.crt"');
  });

  it("node snippet uses ProxyAgent with requestTls.ca", () => {
    const js = makeSnippet("node", "host");
    expect(js).toContain('import { ProxyAgent, fetch } from "undici"');
    expect(js).toContain("requestTls:");
    expect(js).toContain('readFileSync("./proxymetrics-ca.crt")');
  });

  it("go snippet builds an http.Client with RootCAs", () => {
    const go = makeSnippet("go", "host");
    expect(go).toContain("base64.RawURLEncoding.EncodeToString");
    expect(go).toContain("x509.NewCertPool()");
    expect(go).toContain("http.ProxyURL(proxyURL)");
  });

  it("ruby snippet uses Net::HTTP with cert_store", () => {
    const rb = makeSnippet("ruby", "host");
    expect(rb).toContain("require \"net/http\"");
    expect(rb).toContain("Base64.urlsafe_encode64");
    expect(rb).toContain('add_file("./proxymetrics-ca.crt")');
  });
});
