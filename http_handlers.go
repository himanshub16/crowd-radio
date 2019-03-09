package main

// this file contains implementation of HTTP handlers - REST API

import (
	"github.com/dgrijalva/jwt-go"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"log"
	"net/http"
	"time"
)

var (
	jwtSecret = []byte("secret")
	service   Service
)

func NewHTTPRouter(_service Service) *echo.Echo {
	service = _service

	router := echo.New()
	router.GET("/health", healthCheckHandler)
	router.POST("/login", loginHandler)

	links := router.Group("/link")
	links.Use(middleware.JWT(jwtSecret))
	{
		links.POST("/new", newLinkHandler)
		links.POST("/upvote", upvoteLinkHandler)
		links.POST("/downvote", downvoteLinkHandler)
	}

	radio := router.Group("/radio")
	radio.Use(middleware.JWT(jwtSecret))
	{
		radio.POST("/now_playing", healthCheckHandler)
	}

	return router
}

func loginHandler(c echo.Context) error {
	u := User{}
	if err := c.Bind(&u); err != nil {
		return err
	}
	service.CreateOrUpdateUser(u)

	token := jwt.New(jwt.SigningMethodHS256)
	claims := token.Claims.(jwt.MapClaims)
	claims["user_id"] = u.UserID
	claims["exp"] = time.Now().Add(time.Hour * 72).Unix()
	t, err := token.SignedString(jwtSecret)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, echo.Map{
		"token": t,
	})
}

func newLinkHandler(c echo.Context) error {
	form := struct {
		URL         string `form:"url" validate:"required"`
		DedicatedTo string `form:"dedicated_to"`
	}{}
	if err := c.Bind(&form); err != nil {
		return c.String(http.StatusBadRequest, "Missing form data")
	}
	log.Println(form.URL, "is the url")
	userID := getUserIDFromContext(c)

	link, err := service.SubmitLink(form.URL, userID, form.DedicatedTo)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"message": err.Error(),
		})
	}
	return c.JSON(http.StatusOK, link)
}

func downvoteLinkHandler(c echo.Context) error {
	form := struct {
		LinkID int64 `form:"link_id" validate:"required"`
	}{}
	if err := c.Bind(&form); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"message": "Missing link_id",
		})
	}
	userID := getUserIDFromContext(c)
	service.Vote(form.LinkID, userID, -1)
	return c.JSON(http.StatusOK, echo.Map{
		"message": "Done",
	})
}

func upvoteLinkHandler(c echo.Context) error {
	form := struct {
		LinkID int64 `form:"link_id" validate:"required"`
	}{}
	if err := c.Bind(&form); err != nil || form.LinkID == 0 {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"message": "Missing link_id",
		})
	}
	userID := getUserIDFromContext(c)
	log.Println(form.LinkID)
	service.Vote(form.LinkID, userID, +1)
	return c.JSON(http.StatusOK, echo.Map{
		"message": "Done",
	})
}

func healthCheckHandler(c echo.Context) error {
	message := c.QueryParam("message")
	service.Test(message)
	return c.String(http.StatusOK, "I am up and running!")
}

func getUserIDFromContext(c echo.Context) string {
	return c.Get("user").(*jwt.Token).Claims.(jwt.MapClaims)["user_id"].(string)
}
