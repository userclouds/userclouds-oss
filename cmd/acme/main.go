package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/service/acm"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	elbTypes "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
	"github.com/hlandau/acmeapi"

	"userclouds.com/infra/logtransports"
	"userclouds.com/infra/ucaws"
	"userclouds.com/infra/uclog"
)

// https://datatracker.ietf.org/doc/html/rfc8555#section-8.3

func main() {
	ctx := context.Background()

	// domain := "console.tenant.debug.userclouds.com"
	domain := "auth.poweredbydietcoke.com"
	logtransports.InitLoggerAndTransportsForTools(ctx, uclog.LogLevelDebug, uclog.LogLevelVerbose, "acme")
	defer logtransports.Close()

	fmt.Println("starting")

	rc, err := acmeapi.NewRealmClient(acmeapi.RealmClientConfig{
		// DirectoryURL: "https://acme-staging-v02.api.letsencrypt.org/directory",
		DirectoryURL: "https://acme-v02.api.letsencrypt.org/directory",
	})
	if err != nil {
		uclog.Fatalf(ctx, "Failed to create ACME client: %s", err)
	}

	fmt.Println("created client")

	pk, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		uclog.Fatalf(ctx, "Failed to generate private key: %s", err)
	}

	bs := x509.MarshalPKCS1PrivateKey(pk)
	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	if err := pem.Encode(w, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: bs}); err != nil {
		uclog.Fatalf(ctx, "failed to encode PEM: %v", err)
	}
	w.Flush()
	fmt.Println("Private key:", buf.String())

	// 	b, _ := pem.Decode([]byte(`-----BEGIN RSA PRIVATE KEY-----
	// MIIEowIBAAKCAQEAy1xRSpFG0JAgVZgIo5sW8Zj44b1AGP+T7Q9/X1zXlLOZVgAM
	// FXM5Ol+O+aIyQLGcFfa2wLbIo33Q6KtaXV6y+t25MMtkVYB1xGtSqcfhSNBEL5SA
	// PqVQNm2NIK7UVkMznlTy//5UclfBQZEBJjT9XPNABvfuaxtj4JxmedYMl3eQbaWb
	// qAVcOsCXipmB/TeJdwLLJbD9FRUkkynzXdcd4kiGTZMR3dutA38KtEuMZyv+UhQ7
	// r+ZGGju3FrpmsRsI7wyZy1m1n/2euJcqN3MRcNm8mnV2vs0V1P4qShqm/8IDxGMb
	// Ukhe8x/kIAzVLR4gdnjqMTEm5sh0Dh4yo+RQMwIDAQABAoIBAD1e9OFuClLyy+9I
	// 3IKTUU9D/QgTFv70Um4eWTAsUpr7wHClvv/SMBkfsYRAoK3Ja/Ns6yYpg09jruIo
	// pDK9W4I925+QIg1zoRbP1LiMK77Pq2Q4iqNdPGHQmeCdIlOOIEvOEy+ST2XaeeYR
	// nqkrILMmbdIsjHUiZPfp+zsVj+M9lffeI1H/7kR851kUnYqM8cqeK2Oyj+3/vlC1
	// 2Ao1BTVL/PrpbCaGslAZxkL+ozQ9hSKD08gYP3eEVwtvWHRKiLkwvyJkQco5EDQx
	// QQ6nwRPwtR6E+2fjbmF4Vhu2v/AdLxL0XdSzR9+x36jakHfG3TZeYFbNLYHhaxjs
	// n18ZSUECgYEA9M3Ou4tEob0KT4+xIDX68FU+MMctItfE+PoI2jSJo6+Rn5NInv0m
	// 7TGg1cDoiHLUFXdOXZsRax5jd4JRE/kzA9BUXrei5w+nZXMUNeIKhQnptzm+OAwa
	// GwFHSR/H7+S/n9VHKZiqCIaMeDl+T28Wdrv05tKF8vnkeBBSXVxrjlECgYEA1KlJ
	// b/4PZYybpSy9ygmUZ1SPOSNWUkw6f59kWAqCROr3EosXzKE+xRCKRn4NvW0L9tVS
	// +cGGjUXikfTI89Hvq2XkDckJdn0kq3hjH2lOJZTdUQ54atMgCiqigvyLDPZxHv6E
	// 9YRGs5R/V1zOp3gXSUGgiCgav1dGjPtkb2qDwUMCgYEAhZ0XKvG2gfil+grZiFUu
	// I6LDEOiFUDEohyQhVMe8ICUhfFFtH6nYZznhKQnjYSYbb6Pwl9KdTTQG4iG0kww+
	// teQtSI0+UpMOsKaA72/yge6JK8JOelTQotCt0dGQ1PSrSlekQaXbmE+nt67ZrA1b
	// 2253Gszo41dVRdrSubZJ1iECgYAyUZgZ0sCr45hUfgCuVJPz2zNEbtMXCAhzeDCT
	// EHGAgyRRE+5eseybTm2Zfmwb3TiOgC1xAVSoCtgwdv4xiwQtxx8uD9qYWcYaeJLj
	// tNQT2mR/sG/Xvvr+zNXFLqJsP8fGcKzfNxxPk5yJ/GC9iZHg+JFWhj7F9Y2xAC7k
	// mgxw5wKBgFbuwNP483PNLoPYWS/794IfWVief5HokffcdBFMGlbHLnTmwKkm1lSk
	// Tm+jSnLjW5txR3eB4b2JD3U5SAR+1bod//kZ0i00/UqMzYIlmQNYp52/V3UPLtB2
	// kyETwk/CzvboctuE1BFxnUqm44LZQ+Icp4refH9DFFD2pCiXLBld
	// -----END RSA PRIVATE KEY-----`))
	// 	pk, err := x509.ParsePKCS1PrivateKey(b.Bytes)
	// 	if err != nil {
	// 		uclog.Fatalf(ctx, "failed to parse private key: %v", err)
	// 	}

	// j, err := jwk.New(pk)
	// if err != nil {
	// 	uclog.Fatalf(ctx, "failed to create JWK: %v", err)
	// }

	// https://datatracker.ietf.org/doc/html/rfc7638
	// FIMXE: I think the spec says URL but expects Std b64?
	// https://www.rfc-editor.org/rfc/rfc7515#section-2
	// bs, err := j.Thumbprint(crypto.SHA256)
	// if err != nil {
	// 	uclog.Fatalf(ctx, "failed to create thumbprint: %v", err)
	// }
	// fmt.Println(base64.URLEncoding.EncodeToString(bs))

	acct := acmeapi.Account{
		// URL:                  "https://acme-staging-v02.api.letsencrypt.org/acme/acct/90902943",
		// URL:                  "https://acme-v02.api.letsencrypt.org/acme/acct/989438966",
		PrivateKey:           pk,
		ContactURIs:          []string{"mailto:security@userclouds.com"},
		TermsOfServiceAgreed: true,
	}

	if err := rc.RegisterAccount(ctx, &acct); err != nil {
		uclog.Fatalf(ctx, "Failed to register account: %s", err)
	}

	fmt.Println("Registered account")

	fmt.Println(acct)
	if true {
		os.Exit(0)
	}
	// fmt.Println(rc.GetMeta(ctx))

	// TODO: investigate ACME pre-authorization?
	order := acmeapi.Order{
		// URL: "https://acme-staging-v02.api.letsencrypt.org/acme/order/90902943/7512525374",
		URL: "https://acme-v02.api.letsencrypt.org/acme/order/989438966/167710297996",
		Identifiers: []acmeapi.Identifier{{
			Type:  "dns",
			Value: domain,
		}},
	}

	// if err := rc.NewOrder(ctx, &acct, &order); err != nil {
	// 	uclog.Fatalf(ctx, "Failed to create order: %s", err)
	// }

	if err := rc.LoadOrder(ctx, &acct, &order); err != nil {
		uclog.Fatalf(ctx, "Failed to load order: %s", err)
	}

	fmt.Printf("%+v\n", order)

	az := acmeapi.Authorization{
		URL: order.AuthorizationURLs[0],
	}
	if err := rc.LoadAuthorization(ctx, &acct, &az); err != nil {
		uclog.Fatalf(ctx, "Failed to load authorization: %s", err)
	}

	fmt.Printf("%+v\n", az)

	// var ch acmeapi.Challenge
	// for _, c := range az.Challenges {
	// 	if c.Type == "http-01" {
	// 		ch = c
	// 		break
	// 	}
	// }

	// if err := rc.RespondToChallenge(ctx, &acct, &ch, nil); err != nil {
	// 	uclog.Fatalf(ctx, "Failed to respond to challenge: %s", err)
	// }

	// time.Sleep(15 * time.Second)

	// fmt.Println(ch)

	// keyBytes, err := rsa.GenerateKey(rand.Reader, 2048)
	// if err != nil {
	// 	uclog.Fatalf(ctx, "Failed to generate key: %s", err)
	// }

	privkey := `-----BEGIN RSA PRIVATE KEY-----
MIIEogIBAAKCAQEA2ej0iQsQxzLWIV5njVpB040XKwHRDoUlh9rYP+9CFkCtnugp
c9Zb9Vpbb0lGp9/ZJtSfTWpnvswalZhsCEM/UbR763b0Pcc6gQvS6DgYk8uojhb3
UlXoPqADyZy/wnpm+IECb4rtsXr0AjJRFEPaSA2hb8dcec2bQwOnR95JYsrSAEVk
ryDeJJEdp4YmSDFZcepnsfVwih88BDcp+W2dI6VKpgwtA15SugiMg+Uj/rKpcyyO
hVvgA2YDSDqQ966WWV35TBZ/aOWpJ7ou+FlCeammJuSvoHKKdQCTqmh9mdhv2Yi9
ORvUalLQV2KRi4Cjl2s3YOM397623zCK0RkFzwIDAQABAoIBABx96iNVEA+LpRXd
2xpPV9YKV6Im7afBXwPhaG8LoI96S3lAj9L2jzWIZ/YoFZXzndgG6wFbTU9ULpGq
yU1XRZswRxeliQ/e0dZ3rk7wrr38XgCeHh5k3yX5FCWzWhtal8YcUC43cxbGpcr3
u0Q0DwFmztnnrj661Hcxrhimht/kuR3otF6O2KgoxV5gyUn1QZtHLMcvqRTEzwWQ
IxzdXAlsNOAo8lBfS6dcFLb+BYNSg6uQe+FfwkU3X8FKgSf4rm5BV9n4l0Oxm7k6
/wWKVi8z5Lj8+BAEI7si+OxG3Hjz1PKAIWeKp/ilvMtVrPmm6sv5HxtMaI0Qqadw
YboMb6kCgYEA28obFXtdb56snanoegLWf3skynLYib0TOjBh47qpwOCW+UlYrXUq
5KyO5yqytF7X0u2C/xFFLk1D/2pIWWW+VYzVD/Gyzd90dZ/CBKvNvXaV5lK5NOvT
FmkLtmmzWRzWlX1BFDu/83spMyJvO7LDJZM1Vyy/jCdq4PbtukHhxBMCgYEA/c+U
U/HK7IlBdKI5Jy9SFMWp6xcgeoVKRe6BwNKiA8sR9CQIieH7gLwiFGRg0SgOA6k9
MKJQ9hir8w6qLL213XdOzRyYLVS3ZTfh7jvk0xbpcsjL7+f44IuAlqZ/pMcWpiif
nYcFjdeo1s3OuqKzBcW20IG1rOeu9o7IriFj1tUCgYAdB4SsQa8Fnx+Nc3ORKe7K
x6kEVEblamOvu9QyD+V75C4MnvNndaJEscXuImWYDS7UXSqAJffNNcdVZORJanbJ
NeCuSm4jYvAu2Pr3QvnZnGAQG7z6kGtA+n7hiPR3QKfW9sQxt/KSZiH67wFiESpV
PCw/Z1mlWU90hyi/ARSgoQKBgFaVer9MS/J3PFoigSbJ8NFfQQEO6aiCUf0bSS5T
bKuomd5UcIlBC0A2bdXRDGotpOJA2Lv/k2jwr7AB/7G1ohYD/mDgcVV9gfbIoo1X
507PkSH0OAYGd5N6Y4qBEChRNnvGffUKO63QtStaGDz5BeNhOGVW6ngqrTg1K2aj
w2BlAoGAQ0xi3cA5jA4HQuIBlz4N8JKZ/729cOgwblAJSaDUQyCvNxItLp+YUqdt
ssHblqBPoIrthplNyEyxdjtnmwX47AoxPEkKbiDtTE5MyDjxc5Yfz02hJhwP5RRP
2RqAaK1BQN2uyu4DGz5bs6Lg3n3lf/r/CP5vaYDhKwwq7Gv1BIk=
-----END RSA PRIVATE KEY-----`
	b, _ := pem.Decode([]byte(privkey))
	privCertKey, err := x509.ParsePKCS1PrivateKey(b.Bytes)
	if err != nil {
		uclog.Fatalf(ctx, "failed to parse private key: %v", err)
	}

	bs = x509.MarshalPKCS1PrivateKey(privCertKey)
	// var buf bytes.Buffer
	w = bufio.NewWriter(&buf)
	if err := pem.Encode(w, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: bs}); err != nil {
		uclog.Fatalf(ctx, "failed to encode PEM: %v", err)
	}
	w.Flush()
	fmt.Println("Private key:", buf.String())

	// subj := pkix.Name{
	// 	CommonName:   domain,
	// 	Country:      []string{"US"},
	// 	Province:     []string{"CA"},
	// 	Locality:     []string{"Palo Alto"},
	// 	Organization: []string{"UserClouds"},
	// }
	// rawSubj := subj.ToRDNSequence()
	// asn1Subj, err := asn1.Marshal(rawSubj)
	// if err != nil {
	// 	uclog.Fatalf(ctx, "Failed to marshal subject: %s", err)
	// }

	// tmpl := x509.CertificateRequest{
	// 	RawSubject:         asn1Subj,
	// 	SignatureAlgorithm: x509.SHA256WithRSA,
	// }
	// csr, err := x509.CreateCertificateRequest(rand.Reader, &tmpl, privCertKey)
	// if err != nil {
	// 	uclog.Fatalf(ctx, "Failed to create CSR: %s", err)
	// }

	// fmt.Println(string(csr))

	// if err := rc.Finalize(ctx, &acct, &order, csr); err != nil {
	// 	uclog.Fatalf(ctx, "Failed to finalize order: %s", err)
	// }

	cert := acmeapi.Certificate{
		URL: order.CertificateURL,
	}
	if err := rc.LoadCertificate(ctx, &acct, &cert); err != nil {
		uclog.Fatalf(ctx, "Failed to load certificate: %s", err)
	}

	var actualCert = []byte(`-----BEGIN CERTIFICATE-----
MIIFPDCCBCSgAwIBAgISA13YYE7+psWY31RL6N/M4X3jMA0GCSqGSIb3DQEBCwUA
MDIxCzAJBgNVBAYTAlVTMRYwFAYDVQQKEw1MZXQncyBFbmNyeXB0MQswCQYDVQQD
EwJSMzAeFw0yMzAzMDIwNTM1NDNaFw0yMzA1MzEwNTM1NDJaMCUxIzAhBgNVBAMT
GmF1dGgucG93ZXJlZGJ5ZGlldGNva2UuY29tMIIBIjANBgkqhkiG9w0BAQEFAAOC
AQ8AMIIBCgKCAQEA2ej0iQsQxzLWIV5njVpB040XKwHRDoUlh9rYP+9CFkCtnugp
c9Zb9Vpbb0lGp9/ZJtSfTWpnvswalZhsCEM/UbR763b0Pcc6gQvS6DgYk8uojhb3
UlXoPqADyZy/wnpm+IECb4rtsXr0AjJRFEPaSA2hb8dcec2bQwOnR95JYsrSAEVk
ryDeJJEdp4YmSDFZcepnsfVwih88BDcp+W2dI6VKpgwtA15SugiMg+Uj/rKpcyyO
hVvgA2YDSDqQ966WWV35TBZ/aOWpJ7ou+FlCeammJuSvoHKKdQCTqmh9mdhv2Yi9
ORvUalLQV2KRi4Cjl2s3YOM397623zCK0RkFzwIDAQABo4ICVzCCAlMwDgYDVR0P
AQH/BAQDAgWgMB0GA1UdJQQWMBQGCCsGAQUFBwMBBggrBgEFBQcDAjAMBgNVHRMB
Af8EAjAAMB0GA1UdDgQWBBT9hMj2knwmHWnQvTm/AwNjL4jhUDAfBgNVHSMEGDAW
gBQULrMXt1hWy65QCUDmH6+dixTCxjBVBggrBgEFBQcBAQRJMEcwIQYIKwYBBQUH
MAGGFWh0dHA6Ly9yMy5vLmxlbmNyLm9yZzAiBggrBgEFBQcwAoYWaHR0cDovL3Iz
LmkubGVuY3Iub3JnLzAlBgNVHREEHjAcghphdXRoLnBvd2VyZWRieWRpZXRjb2tl
LmNvbTBMBgNVHSAERTBDMAgGBmeBDAECATA3BgsrBgEEAYLfEwEBATAoMCYGCCsG
AQUFBwIBFhpodHRwOi8vY3BzLmxldHNlbmNyeXB0Lm9yZzCCAQYGCisGAQQB1nkC
BAIEgfcEgfQA8gB3ALc++yTfnE26dfI5xbpY9Gxd/ELPep81xJ4dCYEl7bSZAAAB
hqEIphAAAAQDAEgwRgIhAPBVj9Tvz8JOGKQAlNyJr0qGEBURJeTi6ELrwF4SPKKm
AiEAw19wgah11q8BB/I60ztJz7DJa+k+L2m6UE7SeHZI6SMAdwDoPtDaPvUGNTLn
Vyi8iWvJA9PL0RFr7Otp4Xd9bQa9bgAAAYahCKYRAAAEAwBIMEYCIQDthJL4anxq
78tfM8e9g0VV1wTRLgLpmwf3lZrVSJROygIhAPn+VaSK1Ksfwe/BVNOJYHgGQPpR
lta8JH/lQ8HUbsHMMA0GCSqGSIb3DQEBCwUAA4IBAQBZCyLDc7acw35YyAVfVfY+
pgWTlJ0QVdtuX7NtmtVsmuypLIT65vsgd6tmiuFjO37uFFpJqItF/qCQfZNTqwAi
o5DG2cFyV6rCbUuF8FIkOdvhPmgQhMMBNn/aNfwOc8Zf+jY48rmkkY6u3xdMam2P
NWhwopFjFJVrtZupYUqaw7itLo+ctMYer1AzqvHBUcOR7pIAE7PRdKnvoSBedpMx
6E8iRfqWSYK+BvPExJFwasXwHC7m8GQ99eikuCHr94qnZ16tTCaMw9r0zGZzvlM3
wVvTmbtVdRR5iOOrNzHbDMzB1504PAAQ24mE1S1FrURWldL9Zrc0NMBARGy2w3eN
-----END CERTIFICATE-----`)

	var certChain = []byte(`-----BEGIN CERTIFICATE-----
MIIFFjCCAv6gAwIBAgIRAJErCErPDBinU/bWLiWnX1owDQYJKoZIhvcNAQELBQAw
TzELMAkGA1UEBhMCVVMxKTAnBgNVBAoTIEludGVybmV0IFNlY3VyaXR5IFJlc2Vh
cmNoIEdyb3VwMRUwEwYDVQQDEwxJU1JHIFJvb3QgWDEwHhcNMjAwOTA0MDAwMDAw
WhcNMjUwOTE1MTYwMDAwWjAyMQswCQYDVQQGEwJVUzEWMBQGA1UEChMNTGV0J3Mg
RW5jcnlwdDELMAkGA1UEAxMCUjMwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEK
AoIBAQC7AhUozPaglNMPEuyNVZLD+ILxmaZ6QoinXSaqtSu5xUyxr45r+XXIo9cP
R5QUVTVXjJ6oojkZ9YI8QqlObvU7wy7bjcCwXPNZOOftz2nwWgsbvsCUJCWH+jdx
sxPnHKzhm+/b5DtFUkWWqcFTzjTIUu61ru2P3mBw4qVUq7ZtDpelQDRrK9O8Zutm
NHz6a4uPVymZ+DAXXbpyb/uBxa3Shlg9F8fnCbvxK/eG3MHacV3URuPMrSXBiLxg
Z3Vms/EY96Jc5lP/Ooi2R6X/ExjqmAl3P51T+c8B5fWmcBcUr2Ok/5mzk53cU6cG
/kiFHaFpriV1uxPMUgP17VGhi9sVAgMBAAGjggEIMIIBBDAOBgNVHQ8BAf8EBAMC
AYYwHQYDVR0lBBYwFAYIKwYBBQUHAwIGCCsGAQUFBwMBMBIGA1UdEwEB/wQIMAYB
Af8CAQAwHQYDVR0OBBYEFBQusxe3WFbLrlAJQOYfr52LFMLGMB8GA1UdIwQYMBaA
FHm0WeZ7tuXkAXOACIjIGlj26ZtuMDIGCCsGAQUFBwEBBCYwJDAiBggrBgEFBQcw
AoYWaHR0cDovL3gxLmkubGVuY3Iub3JnLzAnBgNVHR8EIDAeMBygGqAYhhZodHRw
Oi8veDEuYy5sZW5jci5vcmcvMCIGA1UdIAQbMBkwCAYGZ4EMAQIBMA0GCysGAQQB
gt8TAQEBMA0GCSqGSIb3DQEBCwUAA4ICAQCFyk5HPqP3hUSFvNVneLKYY611TR6W
PTNlclQtgaDqw+34IL9fzLdwALduO/ZelN7kIJ+m74uyA+eitRY8kc607TkC53wl
ikfmZW4/RvTZ8M6UK+5UzhK8jCdLuMGYL6KvzXGRSgi3yLgjewQtCPkIVz6D2QQz
CkcheAmCJ8MqyJu5zlzyZMjAvnnAT45tRAxekrsu94sQ4egdRCnbWSDtY7kh+BIm
lJNXoB1lBMEKIq4QDUOXoRgffuDghje1WrG9ML+Hbisq/yFOGwXD9RiX8F6sw6W4
avAuvDszue5L3sz85K+EC4Y/wFVDNvZo4TYXao6Z0f+lQKc0t8DQYzk1OXVu8rp2
yJMC6alLbBfODALZvYH7n7do1AZls4I9d1P4jnkDrQoxB3UqQ9hVl3LEKQ73xF1O
yK5GhDDX8oVfGKF5u+decIsH4YaTw7mP3GFxJSqv3+0lUFJoi5Lc5da149p90Ids
hCExroL1+7mryIkXPeFM5TgO9r0rvZaBFOvV2z0gp35Z0+L4WPlbuEjN/lxPFin+
HlUjr8gRsI3qfJOQFy/9rKIJR0Y/8Omwt/8oTWgy1mdeHmmjk7j1nYsvC9JSQ6Zv
MldlTTKB3zhThV1+XWYp6rjd5JW1zbVWEkLNxE7GJThEUG3szgBVGP7pSWTUTsqX
nLRbwHOoq7hHwg==
-----END CERTIFICATE-----
-----BEGIN CERTIFICATE-----
MIIFYDCCBEigAwIBAgIQQAF3ITfU6UK47naqPGQKtzANBgkqhkiG9w0BAQsFADA/
MSQwIgYDVQQKExtEaWdpdGFsIFNpZ25hdHVyZSBUcnVzdCBDby4xFzAVBgNVBAMT
DkRTVCBSb290IENBIFgzMB4XDTIxMDEyMDE5MTQwM1oXDTI0MDkzMDE4MTQwM1ow
TzELMAkGA1UEBhMCVVMxKTAnBgNVBAoTIEludGVybmV0IFNlY3VyaXR5IFJlc2Vh
cmNoIEdyb3VwMRUwEwYDVQQDEwxJU1JHIFJvb3QgWDEwggIiMA0GCSqGSIb3DQEB
AQUAA4ICDwAwggIKAoICAQCt6CRz9BQ385ueK1coHIe+3LffOJCMbjzmV6B493XC
ov71am72AE8o295ohmxEk7axY/0UEmu/H9LqMZshftEzPLpI9d1537O4/xLxIZpL
wYqGcWlKZmZsj348cL+tKSIG8+TA5oCu4kuPt5l+lAOf00eXfJlII1PoOK5PCm+D
LtFJV4yAdLbaL9A4jXsDcCEbdfIwPPqPrt3aY6vrFk/CjhFLfs8L6P+1dy70sntK
4EwSJQxwjQMpoOFTJOwT2e4ZvxCzSow/iaNhUd6shweU9GNx7C7ib1uYgeGJXDR5
bHbvO5BieebbpJovJsXQEOEO3tkQjhb7t/eo98flAgeYjzYIlefiN5YNNnWe+w5y
sR2bvAP5SQXYgd0FtCrWQemsAXaVCg/Y39W9Eh81LygXbNKYwagJZHduRze6zqxZ
Xmidf3LWicUGQSk+WT7dJvUkyRGnWqNMQB9GoZm1pzpRboY7nn1ypxIFeFntPlF4
FQsDj43QLwWyPntKHEtzBRL8xurgUBN8Q5N0s8p0544fAQjQMNRbcTa0B7rBMDBc
SLeCO5imfWCKoqMpgsy6vYMEG6KDA0Gh1gXxG8K28Kh8hjtGqEgqiNx2mna/H2ql
PRmP6zjzZN7IKw0KKP/32+IVQtQi0Cdd4Xn+GOdwiK1O5tmLOsbdJ1Fu/7xk9TND
TwIDAQABo4IBRjCCAUIwDwYDVR0TAQH/BAUwAwEB/zAOBgNVHQ8BAf8EBAMCAQYw
SwYIKwYBBQUHAQEEPzA9MDsGCCsGAQUFBzAChi9odHRwOi8vYXBwcy5pZGVudHJ1
c3QuY29tL3Jvb3RzL2RzdHJvb3RjYXgzLnA3YzAfBgNVHSMEGDAWgBTEp7Gkeyxx
+tvhS5B1/8QVYIWJEDBUBgNVHSAETTBLMAgGBmeBDAECATA/BgsrBgEEAYLfEwEB
ATAwMC4GCCsGAQUFBwIBFiJodHRwOi8vY3BzLnJvb3QteDEubGV0c2VuY3J5cHQu
b3JnMDwGA1UdHwQ1MDMwMaAvoC2GK2h0dHA6Ly9jcmwuaWRlbnRydXN0LmNvbS9E
U1RST09UQ0FYM0NSTC5jcmwwHQYDVR0OBBYEFHm0WeZ7tuXkAXOACIjIGlj26Ztu
MA0GCSqGSIb3DQEBCwUAA4IBAQAKcwBslm7/DlLQrt2M51oGrS+o44+/yQoDFVDC
5WxCu2+b9LRPwkSICHXM6webFGJueN7sJ7o5XPWioW5WlHAQU7G75K/QosMrAdSW
9MUgNTP52GE24HGNtLi1qoJFlcDyqSMo59ahy2cI2qBDLKobkx/J3vWraV0T9VuG
WCLKTVXkcGdtwlfFRjlBz4pYg1htmf5X6DYO8A4jqv2Il9DjXA6USbW1FzXSLr9O
he8Y4IWS6wY7bCkjCWDcRQJMEhg76fsO3txE+FiYruq9RUWhiF1myv4Q6W+CyBFC
Dfvp7OOGAN6dEOM4+qR9sdjoSYKEBpsr6GtPAQw4dy753ec5
-----END CERTIFICATE-----`)

	// for i, c := range cert.CertificateChain {
	// 	crt, err := x509.ParseCertificate(c)
	// 	if err != nil {
	// 		uclog.Fatalf(ctx, "Failed to parse PK: %s", err)
	// 	}

	// 	var buf bytes.Buffer
	// 	w := bufio.NewWriter(&buf)
	// 	if err := pem.Encode(w, &pem.Block{Type: "CERTIFICATE", Bytes: crt.Raw}); err != nil {
	// 		uclog.Fatalf(ctx, "failed to encode PEM: %v", err)
	// 	}
	// 	w.Flush()
	// 	if i == 0 {
	// 		actualCert = buf.Bytes()
	// 	} else if i == 1 {
	// 		certChain = buf.Bytes()
	// 	} else {
	// 		certChain = append(certChain, buf.Bytes()...)
	// 		uclog.Warningf(ctx, "Certificate chain has more than 1 certificate")
	// 	}

	// 	fmt.Println("Certificate:", buf.String())
	// }

	awsCfg, err := ucaws.NewConfigWithDefaultRegion(ctx)
	if err != nil {
		uclog.Fatalf(ctx, "Failed to create AWS session: %s", err)
	}

	acmc := acm.NewFromConfig(awsCfg)
	ico, err := acmc.ImportCertificate(ctx, &acm.ImportCertificateInput{
		Certificate:      actualCert,
		CertificateChain: certChain,
		PrivateKey:       []byte(privkey),
	})
	if err != nil {
		uclog.Fatalf(ctx, "Failed to import certificate: %s", err)
	}

	fmt.Printf("%+v", ico)

	// debug 443
	la := "arn:aws:elasticloadbalancing:us-west-2:323439664763:listener/app/awseb-AWSEB-1RF8SOE6DTCNY/60e99368c178194c/4632db4a974ebe0f"

	elbc := elasticloadbalancingv2.NewFromConfig(awsCfg)
	alco, err := elbc.AddListenerCertificates(ctx, &elasticloadbalancingv2.AddListenerCertificatesInput{
		Certificates: []elbTypes.Certificate{{CertificateArn: ico.CertificateArn}},
		ListenerArn:  &la,
	})
	if err != nil {
		uclog.Fatalf(ctx, "Failed to add certificate: %s", err)
	}

	fmt.Printf("%+v", alco)
}
