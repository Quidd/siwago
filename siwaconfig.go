package siwago

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io/ioutil"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"time"
)

//aud The audience registered claim key, the value of which identifies the recipient the JWT is intended for.
//Since this token is meant for Apple, use https://appleid.apple.com.
//https://developer.apple.com/documentation/signinwithapplerestapi/generate_and_validate_tokens
const AUD = "https://appleid.apple.com"
const APPLE_AUTH_URL = "https://appleid.apple.com/auth/token"

//struct for JWT Header
type JWTHeader struct {
	Alg string `json:"alg"`
	Kid string `json:"kid"`
}

//struct for JWT Body
type JWTBody struct {
	Iss string `json:"iss"`
	Iat int64  `json:"iat"`
	Exp int64  `json:"exp"`
	Aud string `json:"aud"`
	Sub string `json:"sub"`
}

//struct holding various values needed to generate tokens.
//this should only needed to be initialized once and then can be kept in memory
type SiwaConfig struct {
	KeyId           string        //key Id from Certificates, Identifiers & Profiles on developers.apple.com
	TokenDelta      time.Duration //duration for which you would want the generated client_secret jwt token to be valid, cant be more than 6 months
	TeamId          string        //Team Id that is configured with Key, can also ne found in Certificates, Identifiers & Profiles on developers.apple.com
	BundleId        string        //bundleId for product com.companyname.product
	PemFileContents []byte        //contents of the p8 file
	Nonce           string        //nonce is set while making the request to generate authorization_code. If you dont use it, keep it an empty string
}

//helper function to get SiwaConfig object
func GetObject(keyId string, teamId string, bundleId string, d time.Duration, nonce string) *SiwaConfig {
	return &SiwaConfig{KeyId: keyId, TokenDelta: d, TeamId: teamId, BundleId: bundleId, Nonce: nonce}
}

//function to validate the object
func (self *SiwaConfig) ValidateObject() (bool, error) {

	var errorString string = ""
	if self.KeyId == "" {
		errorString = errorString + " KeyId not set"
	}
	if self.TokenDelta == 0 {
		errorString = errorString + " token exipry set to 0 seconds"
	}
	if self.TeamId == "" {
		errorString = errorString + " TeamId not set"
	}
	if self.BundleId == "" {
		errorString = errorString + " BundleId not set"
	}
	if len(self.PemFileContents) == 0 {
		errorString = errorString + " PemFile not set. Use SetSecretP8File or SetSecretP8String or SetSecretP8Bytes to set it"
	}

	if errorString != "" {
		return false, errors.New(errorString)
	}
	return true, nil
}

//helper function to set secrets value by filename
//the function expects full path to the p8 file generated
//in the keys and certificates section of developer account
//it should look like this:
//-----BEGIN PRIVATE KEY-----
//jkfweshjdjkhjsbjvguybjebvuewkvbbhj+jbdhbjhbvjhbvjhbvbjvbvjvagcve
//jkfweshjdjkhjsbjvguybje/vuewkvbbhjdjbdhbjhbvjhbvjhbvbjvbvjvagcve
//jkfweshjdjkhjsbjvguybjebvuewkvbbhj+jbdhbjhbvjhbvjhbvbjvbvjvagcve
//jkfweshj
//-----END PRIVATE KEY-----
func (self *SiwaConfig) SetSecretP8File(p8Filename string) error {

	var content []byte
	var err error

	content, err = ioutil.ReadFile(p8Filename)
	if err != nil {
		return err
	}
	self.PemFileContents = content
	return nil
}

//helper function to set secret file contents as a string
//this needs to be pem encoded PKCS8 private key
//same format as the p8 file downloaded from apple
//-----BEGIN PRIVATE KEY-----
//jkfweshjdjkhjsbjvguybjebvuewkvbbhj+jbdhbjhbvjhbvjhbvbjvbvjvagcve
//jkfweshjdjkhjsbjvguybje/vuewkvbbhjdjbdhbjhbvjhbvjhbvbjvbvjvagcve
//jkfweshjdjkhjsbjvguybjebvuewkvbbhj+jbdhbjhbvjhbvjhbvbjvbvjvagcve
//jkfweshj
//-----END PRIVATE KEY-----
func (self *SiwaConfig) SetSecretP8String(p8Contents string) {
	self.PemFileContents = []byte(p8Contents)
}

//helper function to set secret file contents as bytes
//this needs to be pem encoded PKCS8 private key
//same format as the p8 file downloaded from apple
//-----BEGIN PRIVATE KEY-----
//jkfweshjdjkhjsbjvguybjebvuewkvbbhj+jbdhbjhbvjhbvjhbvbjvbvjvagcve
//jkfweshjdjkhjsbjvguybje/vuewkvbbhjdjbdhbjhbvjhbvjhbvbjvbvjvagcve
//jkfweshjdjkhjsbjvguybjebvuewkvbbhj+jbdhbjhbvjhbvjhbvbjvbvjvagcve
//jkfweshj
//-----END PRIVATE KEY-----
func (self *SiwaConfig) SetSecretP8Bytes(p8Contents []byte) {
	self.PemFileContents = p8Contents
}

//function to get encoded jwt header
func (self *SiwaConfig) GetEncodedJwtHeader(keyId string) (string, error) {

	var jwtHeader JWTHeader
	var err error
	var jwtHeaderJsonB []byte
	var jwtHeaderBase64Url string

	jwtHeader.Alg = "ES256"
	jwtHeader.Kid = keyId

	jwtHeaderJsonB, err = json.Marshal(jwtHeader)
	if err != nil {
		return "", err
	}
	jwtHeaderBase64Url = base64UrlEncode(jwtHeaderJsonB)
	return jwtHeaderBase64Url, nil
}

//function to get encoded jwt body
func (self *SiwaConfig) GetEncodedJwtBody(bundleId string, teamId string, d time.Duration) (string, error) {

	var jwtBody JWTBody
	var err error
	var jwtBodyJsonB []byte
	var jwtBodyBase64Url string

	jwtBody.Iss = teamId
	jwtBody.Iat = time.Now().Unix()
	jwtBody.Exp = time.Now().Add(d).Unix()
	jwtBody.Aud = AUD
	jwtBody.Sub = bundleId

	jwtBodyJsonB, err = json.Marshal(jwtBody)
	if err != nil {
		return "", err
	}

	jwtBodyBase64Url = base64UrlEncode(jwtBodyJsonB)
	return jwtBodyBase64Url, nil
}

//get the client_secret
func (self *SiwaConfig) GetClientSecret() (string, error) {

	var err error
	var encodedHeader, encodedBody, data, ecdsaHash, clientSecret string
	var hash [32]byte
	var privKey *ecdsa.PrivateKey
	var r, s *big.Int
	var hashBytes []byte

	if _, err = self.ValidateObject(); err != nil {
		return "", err
	}

	//get encoded heaader
	encodedHeader, err = self.GetEncodedJwtHeader(self.KeyId)
	if err != nil {
		return "", errors.New("Error while encoding JWT header. " + err.Error())
	}
	//get encoded body
	encodedBody, err = self.GetEncodedJwtBody(self.BundleId, self.TeamId, self.TokenDelta)
	if err != nil {
		return "", errors.New("Error while encoding JWT body. " + err.Error())
	}
	data = encodedHeader + "." + encodedBody
	//compute sha256
	hash = sha256.Sum256([]byte(data))

	//get the private key object
	privKey, err = getPrivKey(self.PemFileContents)
	if err != nil {
		return "", errors.New("Error while generating private key, check P8 File. " + err.Error())
	}

	//sign using the private key
	r, s, err = ecdsa.Sign(rand.Reader, privKey, hash[:])
	if err != nil {
		return "", errors.New("Error while signing. " + err.Error())
	}

	//join r and s
	hashBytes = append(r.Bytes(), s.Bytes()...)
	//base64urlencode  the bytes
	ecdsaHash = base64UrlEncode(hashBytes)

	//secret is <base64url(jsonHeader)>"."<base64url(jsonBody)>"."<signed hash>
	clientSecret = data + "." + ecdsaHash

	return clientSecret, nil
}

//put together the data to make a request to apple
//and return the generated token as an object
func (self *SiwaConfig) ExchangeAuthCode(code string, redirectUri string) (*Token, error) {

	var err error
	var clientSecret string
	var form url.Values
	var c http.Client
	var req *http.Request
	var resp *http.Response
	var bodyContents []byte
	var tok Token
	var reason string

	//check if siwa object is valid, all required values have been set
	if _, err = self.ValidateObject(); err != nil {
		return nil, err
	}

	//gather form values for post
	clientSecret, err = self.GetClientSecret()
	if err != nil {
		return nil, errors.New("Error while generating client_secret. " + err.Error())
	}
	form = url.Values{}
	form.Add("client_id", self.BundleId)
	form.Add("client_secret", clientSecret)
	form.Add("code", code)
	form.Add("grant_type", "authorization_code")
	form.Add("redirect_uri", redirectUri)

	//initiate the http request
	c = http.Client{Timeout: 5 * time.Second}
	req, err = http.NewRequest("POST", APPLE_AUTH_URL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err = c.Do(req)
	if err != nil {
		return nil, err
	}

	//read response
	bodyContents, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	//extract into an object
	err = json.Unmarshal(bodyContents, &tok)
	if err != nil {
		return nil, err
	}

	//check if there was an error with the request
	if tok.Error != "" {
		return &tok, errors.New(tok.Error)
	}

	//validate id token
	tok.Valid, reason = ValidateIdTokenWithNonce(self.BundleId, tok.IdToken, self.Nonce)
	if !tok.Valid {
		//if invalid, add message as an error
		return &tok, errors.New(reason)
	}

	return &tok, nil
}
