package main

// this file contains implementation of HTTP handlers - REST API

import (
	"github.com/dgrijalva/jwt-go"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"log"
	"net/http"
	"strconv"
	"time"
)

var (
	jwtSecret = []byte("secret")
	service   Service
	radio     *Radio
)

func NewHTTPRouter(_service Service, _radio *Radio) *echo.Echo {
	service = _service
	radio = _radio

	r := echo.New()
	r.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: "method=${method}, uri=${uri}, status=${status}\n",
	}))
	// router := echo.New()
	router := r.Group("/api")
	router.GET("/health", healthCheckHandler)
	router.POST("/login", loginHandler)

	linkGroup := router.Group("/link")
	linkGroup.Use(middleware.JWT(jwtSecret))
	{
		linkGroup.GET("/:id", linkByIdHandler)
		linkGroup.GET("/by_me", linksByMeHandler)
		linkGroup.POST("/new", newLinkHandler)
		linkGroup.POST("/upvote", upvoteLinkHandler)
		linkGroup.POST("/downvote", downvoteLinkHandler)
	}

	radioGroup := router.Group("/radio")
	radioGroup.Use(middleware.JWT(jwtSecret))
	{
		radioGroup.GET("/now_playing", radioGetNowPlayingHandler)
		radioGroup.GET("/queue", radioGetQueueHandler)
	}

	// return router
	return r
}

func linkByIdHandler(c echo.Context) error {
	lid, _ := strconv.Atoi(c.Param("id"))
	l, _ := service.GetLinkByID(int64(lid))
	return c.JSON(http.StatusOK, l)
}

func linksByMeHandler(c echo.Context) error {
	userID := getUserIDFromContext(c)
	links := service.GetLinksByUser(userID)
	return c.JSON(http.StatusOK, links)
}

func loginHandler(c echo.Context) error {
	u := User{}
	if err := c.Bind(&u); err != nil {
		return err
	}
	// I shouldn't be doing this
	u.UserID = c.FormValue("user_id")
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

func radioGetNowPlayingHandler(c echo.Context) error {
	if radio.nowPlaying == nil {
		return c.JSON(http.StatusOK, echo.Map{
			"state":       "idle",
			"link":        nil,
			"player_time": 0,
		})
	}

	userID := getUserIDFromContext(c)
	links := []Link{*radio.nowPlaying}
	votes := service.GetVotesForUser(links, userID)

	linkIDs := make([]int64, len(links))
	for i, l := range links {
		linkIDs[i] = l.LinkID
	}

	totalVotes := service.GetTotalVoteForLinks(linkIDs)

	for i, l := range links {
		if score, ok := totalVotes[l.LinkID]; ok {
			links[i].TotalVotes = score
		} else {
			links[i].TotalVotes = 0
		}
	}

	var myVote int64
	if len(votes) == 0 {
		myVote = 0
	} else {
		myVote = votes[radio.nowPlaying.LinkID]
	}

	user := service.GetUserByID(radio.nowPlaying.SubmittedBy)

	return c.JSON(http.StatusOK, echo.Map{
		"state": "running",
		"link":  radio.nowPlaying,
		"submitted_by": echo.Map{
			"firstname": user.FirstName,
			"lastname":  user.LastName,
		},
		"my_vote":     myVote,
		"player_time": radio.playerCurTimeSec,
	})
}

func radioGetQueueHandler(c echo.Context) error {
	links := radio.queue
	userID := getUserIDFromContext(c)
	votes := service.GetVotesForUser(links, userID)

	for i, l := range links {
		if vote, ok := votes[l.LinkID]; ok {
			links[i].MyVote = vote
		} else {
			links[i].MyVote = 0
		}
	}

	return c.JSON(http.StatusOK, echo.Map{
		"links": links,
		"votes": votes,
	})
}
