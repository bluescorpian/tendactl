package tenda

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"sync"

	"github.com/manifoldco/promptui"
)

const Filename = "tendactl-session-password.txt"

func SetSessionPassword(password string) error {
	tempDir := os.TempDir()
	filepath := filepath.Join(tempDir, Filename)

	f, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(password)
	if err != nil {
		return err
	}

	return nil
}

func GetSessionPassword() string {
	tempDir := os.TempDir()
	filepath := filepath.Join(tempDir, Filename)

	f, err := os.Open(filepath)
	if err != nil {
		return ""
	}
	defer f.Close()

	dat, err := io.ReadAll(f)
	if err != nil {
		return ""
	}

	return string(dat)
}

func SessionPasswordExists() bool {
	sessionPassword := GetSessionPassword()
	return sessionPassword != ""
}

func GeneratePasswordHash(password string) string {
	hash := md5.New()
	hash.Write([]byte(password))
	hashBytes := hash.Sum(nil)

	hashHex := hex.EncodeToString(hashBytes)
	return hashHex
}

// func EnsurePasswordHashSet() bool {
// 	if (!PasswordHashExists()) {
// 		templates := &promptui.PromptTemplates{
// 			Prompt:  "{{ . }} ",
// 			Valid:   "{{ . }} ",
// 			Invalid: "{{ . | red }} ",
// 			Success: "{{ . | bold }} ",
// 		}
// 		prompt := promptui.Prompt{ Label: "Enter password:", Mask: '*', Templates: templates }
// 		password, err := prompt.Run()
// 		if err != nil {
// 			return false;
// 		}

// 		passwordHash := GeneratePasswordHash(password)
// 		SetPasswordHash(passwordHash)
// 		return true;
// 	} else {
// 		return true;
// 	}
// }

func Login(client *http.Client, passwordHash string) error {
	req, err := TendaRequest("GET", "/login.html", nil)
	if err != nil {
		return err
	}
	_, err = client.Do(req)
	if err != nil {
		return err
	}
	// fmt.Printf("Logging in with password hash: %s\n", passwordHash)
	payload := []byte("username=admin&password=" + passwordHash)
	req, err = TendaRequest("POST", "/login/Auth", bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode == 200 {
		// get response body as string
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		// Convert the response body to a string
		bodyString := string(bodyBytes)
		if bodyString == "1" {
			return fmt.Errorf("password is invalid")
		}

		if resp.Request.URL.String() != "http://192.168.0.1/main.html" {
			return fmt.Errorf("Login failed. Redirected to: %s", resp.Request.URL.String())
		}
		return nil
	} else {
		return fmt.Errorf("Login failed. Status code: %d", resp.StatusCode)
	}
}

func RequestPasswordHash() (string, error) {
	templates := &promptui.PromptTemplates{
		Prompt:  "{{ . }} ",
		Valid:   "{{ . }} ",
		Invalid: "{{ . | red }} ",
		Success: "{{ . }} ",
	}
	prompt := promptui.Prompt{Label: "Enter password:", Mask: '*', Templates: templates}
	password, err := prompt.Run()
	if err != nil {
		os.Exit(1)
	}

	passwordHash := GeneratePasswordHash(password)
	return passwordHash, nil
}

func RefreshSessionPassword(client *http.Client) error {
	passwordHash, err := RequestPasswordHash() // maybe bad? cli code in library code, potential fix, function passed in to request password on a whim.
	if err != nil {
		return err
	}
	err = Login(client, passwordHash)
	if err != nil {
		return err
	}
	// fmt.Printf("Session password is: %s\n", GetSessionPassword())

	return nil
}

func TendaDoAuthRequest(client *http.Client, req *http.Request) (*http.Response, error) {
	sessionPassword := GetSessionPassword()
	if sessionPassword == "" {
		err := RefreshSessionPassword(client)
		if err != nil {
			return nil, err
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return resp, err
	}
	if resp.Request.URL.String() != req.URL.String() && resp.Request.URL.String() == "http://192.168.0.1/login.html" {
		err := RefreshSessionPassword(client)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Cookie", "password="+GetSessionPassword())
		resp, err = client.Do(req)
		if err != nil {
			return nil, err
		}
	}
	return resp, nil
}

type AuthJar struct {
	jar *cookiejar.Jar
	mu  sync.Mutex
}

func NewAuthJar() (*AuthJar, error) {
	j, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	authJar := &AuthJar{jar: j}
	authJar.loadSessionPassword()

	return authJar, nil
}

// SetCookies overrides the default method and adds a lock for concurrency safety
func (c *AuthJar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.jar.SetCookies(u, cookies)
	c.saveSessionPassword()
}

// Cookies returns the cookies for the given URL
func (c *AuthJar) Cookies(u *url.URL) []*http.Cookie {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.jar.Cookies(u)
}

func (c *AuthJar) loadSessionPassword() {
	sessionPassword := GetSessionPassword()

	if sessionPassword == "" {
		return
	}

	cookie, err := http.ParseSetCookie("password=" + sessionPassword + "; path=/")

	if err != nil {
		return
	}

	c.jar.SetCookies(ParsedBaseURL, []*http.Cookie{cookie})
}

func (c *AuthJar) saveSessionPassword() {
	cookies := c.jar.Cookies(ParsedBaseURL)
	for _, cookie := range cookies {
		if cookie.Name == "password" {
			SetSessionPassword(cookie.Value)
			return
		}
	}
}
