package client

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	log "github.com/Sirupsen/logrus"
	"gopkg.in/resty.v0"
)

const (
	defaultAuthURL = "https://token.services.auth.zalando.com/oauth2/access_token" // TODO: use ENV Variable
	defaultScope   = "uid"
	tokenID        = "go-daemon"
)

// ErrData is a not retrieable error, because of corrupted data
var ErrData = errors.New("API: data is corrupt")

// ErrTemporary is a retrieable error, p.e. API throws 500
var ErrTemporary = errors.New("API: temporary error")

// ErrSerious is not a retrieable error, because of unknown issues
var ErrSerious = errors.New("API: Serious error")

// Client is the struct for accessing client functionalities
type Client struct {
	Debug       bool
	URL         string
	Scope       string
	AccessToken string
}

// NewClient returns fully intialized Client
// TODO: make time.Duration configurable
func NewClient(baseURL, token string, debug bool) *Client {
	cli := &Client{
		Debug:       debug,
		URL:         baseURL,
		AccessToken: token,
	}

	go func() {
		for {
			select {
			case <-time.After(10 * time.Minute):
				cli.RefreshToken()
			}

		}
	}()

	return cli
}

// RefreshToken gets a new access token to authenticate requests made
// by the client.
// TODO: get a new AccessToken and set cli.AccessToken to the new one.
func (cli *Client) RefreshToken() error {
	return nil
}

var transport *http.Transport
var restClient *resty.Client

func init() {
	transport = &http.Transport{Dial: (&net.Dialer{
		Timeout: 5 * time.Second,
	}).Dial,
		TLSHandshakeTimeout: 5 * time.Second}
	restClient = resty.New().SetTransport(transport)
	// make DNS failover possible
	go func() {
		for {
			time.Sleep(60 * time.Second)
			transport.CloseIdleConnections()
		}
	}()
}

// SendPUT sends a PUT request with given ID and given body as json to
// the defined URL with appended /<ID> and an http Authorization
// Header Bearer <token>.
func (cli *Client) SendPUT(ID string, body []byte) error {
	now := time.Now()
	urlString := fmt.Sprintf("%s/%v", cli.URL, ID)
	log.Debugf("PUT %v", urlString)
	resp, err := restClient.R().
		SetHeader("Content-Type", "application/json").
		SetBody(body).
		SetAuthToken(cli.AccessToken).
		Put(urlString)
	if err != nil {
		log.Warningf("Could not send PUT request to %v, caused by: %v", urlString, err)
		return ErrTemporary
	}
	log.Printf("Send PUT took %v", time.Now().Sub(now))

	stat := resp.StatusCode()
	switch {
	case 200 <= stat && stat < 300:
		log.Debugf("API - OK: (%s) - for event ID %v", resp.Body(), ID)
	case stat == http.StatusTooManyRequests:
		log.Warningf("Hitting rate limit of API, calm down and retry")
		time.Sleep(time.Second * 3) // TODO think about sleep time and configuration
		return ErrTemporary
	// No schema defined -> OK, not our problem
	case stat == http.StatusNotFound:
		log.Debugf("ID=%v, body=%s", ID, body)
		return nil // TODO: you may want to change this
	case stat == http.StatusUnprocessableEntity:
		log.Printf("Invalid body (%s)", resp.Body())
		return nil // TODO: you may want to change this
	case stat == http.StatusBadRequest:
		log.Errorf("Invalid body (%s)", resp.Body())
		return ErrData
	case stat == http.StatusUnauthorized:
		err := cli.RefreshToken()
		if err != nil {
			log.Errorf("Could not get a new token (will retry later), caused by: %v", err)
			time.Sleep(time.Second * 10) // TODO think about sleep time and configuration
		}
		return ErrTemporary
	case stat == http.StatusForbidden:
		log.Fatal("Your token is not allowed to send data to this API")
	case 400 <= stat && stat < 500:
		log.Errorf("Client failure: %s", resp.Body())
		// TODO: think about what todo, in general for client errors
		return ErrData
	// Server should be healthy in the future, retry
	case 500 <= stat:
		log.Warningf("Server Failure: \nrequest: %s\nresponse: %s", body, resp.Body())
		time.Sleep(time.Second * 10) // TODO think about sleep time and configuration
		return ErrTemporary
	default:
		log.Errorf("Should never happen, response: %s, for data: ID=%v, body=%s", resp.Body(), ID, body)
		time.Sleep(time.Second * 10) // TODO think about sleep time and configuration
		return ErrSerious
	}

	return err
}
