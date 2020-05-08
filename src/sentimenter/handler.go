package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.opencensus.io/trace"
)

func defaultHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"release":      AppVersion,
		"request_on":   time.Now(),
		"request_from": c.Request.RemoteAddr,
	})
}

func scoreHandler(c *gin.Context) {
	ctx, span := trace.StartSpan(c.Request.Context(), "sentimenter-handler")
	defer span.End()

	r := ScoreRequest{}
	if err := c.ShouldBindJSON(&r); err != nil || r.Text == "" {
		logger.Printf("error binding scoring request: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Invalid scoring request format, see logs for details",
		})
		return
	}
	logger.Printf("received scoring request: %v", r)

	// score the content sentiment
	score, err := scoreSentiment(ctx, r.Text, r.Lang)
	if err != nil {
		logger.Printf("error scoring sentiment: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Server Error",
			"message": "Error scoring sentiment, see processor log for details",
		})
		return
	}
	logger.Printf("result: %f - %s", score, r.Text)

	span.Annotate([]trace.Attribute{
		trace.StringAttribute("score", fmt.Sprintf("%f", score)),
		trace.StringAttribute("text", r.Text),
	}, "Processed sentiment")

	c.JSON(http.StatusOK, &SimpleScore{
		Score: score,
		Text:  r.Text,
	})
}

// SimpleScore represents the sentiment score
type SimpleScore struct {
	// Score is the 0 to 1 score of the sentiment
	// (e.g. 0-0.3 bad, 0.3-0.6 neutral, 0.6-1 positive)
	Score float64 `json:"score"`
	// Text is the text to be scored
	Text string `json:"text"`
}

// ScoreRequest represents the input request
type ScoreRequest struct {
	// Text is the text to be scored
	Text string `json:"text"`
	// Lang is the lang of the text
	Lang string `json:"lang"`
}
