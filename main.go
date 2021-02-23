package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

type SlackRequest struct {
	Token          		string `form:"token"`
	TeamId         		string `form:"team_id"`
	TeamDomain     		string `form:"team_domain"`
	EnterpriseId   		string `form:"enterprise_id"`
	EnterpriseName 		string `form:"enterprise_name"`
	ChannelId      		string `form:"channel_id"`
	ChannelName    		string `form:"channel_name"`
	UserId         		string `form:"user_id"`
	UserName       		string `form:"user_name"`
	Command        		string `form:"command"`
	Text           		string `form:"text"`
	ResponseUrl    		string `form:"response_url"`
	TriggerId      		string `form:"trigger_id"`
	ApiAppId       		string `form:"api_app_id"`
	IsEnterpriseInstall bool `form:"is_enterprise_install"`
}

type SlackText struct {
	Type string `json:"type,omitempty"`
	Text string `json:"text"`
}

type SlackImage struct {
	Type     string `json:"type,omitempty"`
	ImageUrl string `json:"image_url"`
	AltText  string `json:"alt_text"`
}

type SlackBlock struct {
	Type 		string `json:"type"`
	Text 		*SlackText `json:"text,omitempty"`
	Accessory 	*SlackImage `json:"accessory,omitempty"`
}

type SlackResponse struct {
	SlackBlocks []SlackBlock `json:"blocks"`
}

var (
	SLACK_SIGNING_SECRET = os.Getenv("SLACK_SIGNING_SECRET")
	DEFAULT_RESPONSE = "Error processing request"
)

func main() {
	log.SetLevel(log.DebugLevel)
	r := gin.Default()
	if len(SLACK_SIGNING_SECRET) == 0 {
		panic("SLACK_SIGNING_SECRET environment variable not specified")
	}

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, "health check response")
	})
	r.POST("/", SlackHandler)

	srv := &http.Server {
		Addr:    ":8080",
		Handler: r,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
	log.Infof("Exiting Server")
}

func SlackHandler(c *gin.Context) {
	requestTimestamp := c.Request.Header.Get("X-Slack-Request-Timestamp")
	requestSignature := c.Request.Header.Get("X-Slack-Signature")
	version := "v0" // Always v0 -- according to Slack docs
	postBytes, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		var response SlackResponse
		log.Errorf("Failed reading body: %s", string(postBytes))
		response.SlackBlocks = append(response.SlackBlocks, SlackBlock{Type: "section", Text: &SlackText{Type: "mrkdwn", Text: DEFAULT_RESPONSE}, Accessory: nil})
		c.JSON(500, response)
		return
	}
	if verifySlackRequest(SLACK_SIGNING_SECRET,version,string(postBytes),requestTimestamp,requestSignature) == true {
		var response SlackResponse
		log.Infof("Success")
		response.SlackBlocks = append(response.SlackBlocks, SlackBlock{Type: "section", Text: &SlackText{Type: "mrkdwn", Text: "Successfully verified request"}, Accessory: nil})
		c.JSON(200, response)
		return
	} else {
		var response SlackResponse
		log.Errorf("Failed verification")
		response.SlackBlocks = append(response.SlackBlocks, SlackBlock{Type: "section", Text: &SlackText{Type: "mrkdwn", Text: "Failed verification"}, Accessory: nil})
		c.JSON(400, response)
		return
	}
}

func verifySlackRequest(signingSecret, version, requestBody, requestTimestamp, requestSignature string) bool {
	stringArray := []string{ version, requestTimestamp, requestBody}
	toVerify := strings.Join(stringArray, ":")
	hash := hmac.New(sha256.New, []byte(signingSecret))
	_, err := io.WriteString(hash, toVerify)
	if err != nil {
		return false
	}
	verificationString := fmt.Sprintf("%s=%x", version, hash.Sum(nil))
	log.Infof("\nHMAC-Sha256: %s\n", verificationString)
	log.Infof("\nVerify Against: %s\n", requestSignature)
	if verificationString == requestSignature {
		return true
	}
	return false
}