package handler

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/chetan-code/gotodo/internal/config"
	"github.com/chetan-code/gotodo/internal/models"
	"github.com/golang-jwt/jwt/v5"
	"github.com/markbates/goth/gothic"
)

type AuthHandler struct {
	config *config.Config
}

func NewAuthHandler(c *config.Config) *AuthHandler {
	return &AuthHandler{config: c}
}

// we are doing this to avoid collision with libraries
type contextKey string

const emailKey contextKey = "userEmail"

func (h *AuthHandler) AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session_token")
		if err != nil {
			HomeRedirect(w, r)
			return
		}

		claims, err := h.VerifyToken(cookie.Value)
		if err != nil {
			HomeRedirect(w, r)
			return
		}

		ctx := context.WithValue(r.Context(), emailKey, claims.Email)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

func (h *AuthHandler) BeginAuth(w http.ResponseWriter, r *http.Request) {
	//gothic look for provide query by default
	//forcing to use google
	q := r.URL.Query()
	q.Add("provider", "google")
	r.URL.RawQuery = q.Encode()

	gothic.BeginAuthHandler(w, r)
}

func (h *AuthHandler) AuthCallbackHandler(w http.ResponseWriter, r *http.Request) {
	user, err := gothic.CompleteUserAuth(w, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//auth success - issue jwt and set cookies
	token, err := h.GenerateJWT(user.Email)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		fmt.Printf("Auth : JWT generation failed : %s", err.Error())
		return
	}

	//token is ready - set cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,              //not visible to JS [IMP for security]
		Secure:   h.config.IsProd(), //enable it for HTTPS in production
	})

	http.Redirect(w, r, "/todos", http.StatusSeeOther)
}

func (h *AuthHandler) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	// clear session cookues
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,              //js cant touch it
		Secure:   h.config.IsProd(), //enable it for HTTPS in production
	})

	//clear gothic session
	gothic.Logout(w, r)
	fmt.Println("Logout success")
	HomeRedirect(w, r)
}

// HELPER FUNCTION
func (h *AuthHandler) GenerateJWT(email string) (string, error) {
	expireTime := time.Now().Add(24 * time.Hour)

	claims := &models.Claims{
		Email: email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expireTime),
		},
	}

	//create the token using hs259 algo
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	//sign with the secret key and return
	return token.SignedString(h.config.GetJWTKey())
}

// HELPER FUNCTION
func (h *AuthHandler) VerifyToken(tokenString string) (*models.Claims, error) {
	claims := &models.Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return h.config.GetJWTKey(), nil
	})

	if err != nil || !token.Valid {
		return nil, err
	}

	return claims, nil
}
