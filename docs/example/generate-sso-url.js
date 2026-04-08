import crypto from "crypto";
const iss = "xx-issuer";
const aud = "statora";
const email = "xxx@email.com";
const secret = "secret";
const baseUrl = "http://localhost:8080";
const
    enc = (obj) => Buffer.from(JSON.stringify(obj)).toString("base64url");
const header = enc({
    alg: "HS256",
    typ: "JWT"
});
const payload = enc({
    iss,
    aud,
    sub: "user-123",
    email,
    exp: Math.floor(Date.now() / 1000) + 3600
});
const signingInput = `${header}.${payload}`;
const
    sig = crypto.createHmac("sha256", secret).update(signingInput).digest("base64url");
const token = `${signingInput}.${sig}`;
console.log(`${baseUrl}/sso/callback?token=${encodeURIComponent(token)}`)