package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/energiegedrag-nl/needforheat-server-api/internal/helpers"
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat"
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/cloudfeed"
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/cloudfeedtype"
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/upload"
	"github.com/energiegedrag-nl/needforheat-server-api/services/cloudfeeds/enelogic"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

const (
	CloudFeedTypeDownloadInterval = time.Hour * 24
)

var (
	ErrDuplicateCloudFeed = errors.New("duplicate cloud feed auth")

	NoLatestUploadTime = needforheat.Time{}
)

type CloudFeedService struct {
	cloudFeedRepo     cloudfeed.CloudFeedRepository
	cloudFeedTypeRepo cloudfeedtype.CloudFeedTypeRepository
	uploadService     *UploadService
	updateChan        chan struct{}
}

// Create a new CloudFeedService.
func NewCloudFeedService(cloudFeedRepo cloudfeed.CloudFeedRepository, cloudFeedTypeRepo cloudfeedtype.CloudFeedTypeRepository, uploadService *UploadService) *CloudFeedService {
	return &CloudFeedService{
		cloudFeedRepo:     cloudFeedRepo,
		cloudFeedTypeRepo: cloudFeedTypeRepo,
		uploadService:     uploadService,
		updateChan:        make(chan struct{}, 1),
	}
}

// Create a new cloudFeed.
// This function exchanges the AuthGrantToken (Code) for a access and refresh token.
func (s *CloudFeedService) Create(ctx context.Context, accountID, cloudFeedTypeID uint, authGrantToken string) (cloudfeed.CloudFeed, error) {
	cloudFeedType, err := s.cloudFeedTypeRepo.Find(cloudfeedtype.CloudFeedType{ID: cloudFeedTypeID})
	if err != nil {
		return cloudfeed.CloudFeed{}, err
	}

	scopes := strings.Split(cloudFeedType.Scope, " ")

	conf := &oauth2.Config{
		ClientID:     cloudFeedType.ClientID,
		ClientSecret: cloudFeedType.ClientSecret,
		Scopes:       scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  cloudFeedType.AuthorizationURL,
			TokenURL: cloudFeedType.TokenURL,
		},
		RedirectURL: cloudFeedType.RedirectURL,
	}

	accessToken, refreshToken, expiry, err := exchangeAuthCode(ctx, conf, authGrantToken)
	if err != nil {
		return cloudfeed.CloudFeed{}, err
	}

	cloudFeed := cloudfeed.MakeCloudFeed(accountID, cloudFeedTypeID, accessToken, refreshToken, needforheat.Time(expiry), authGrantToken)

	cloudFeed, err = s.cloudFeedRepo.Create(cloudFeed)
	if err != nil {
		return cloudFeed, err
	}

	// Signal an update
	s.updateChan <- struct{}{}

	return cloudFeed, nil
}

// Find a cloudFeed using any field set in the cloudFeed struct.
func (s *CloudFeedService) Find(cloudFeed cloudfeed.CloudFeed) (cloudfeed.CloudFeed, error) {
	return s.cloudFeedRepo.Find(cloudFeed)
}

// Refresh the tokens for the CloudFeed corresponding to accountID and cloudFeedTypeID.
func (s *CloudFeedService) RefreshTokens(ctx context.Context, accountID uint, cloudFeedTypeID uint) (cloudfeed.CloudFeed, error) {
	logrus.Infoln("refreshing token for accountID", accountID, "cloudFeedTypeID", cloudFeedTypeID)

	cloudFeed, err := s.cloudFeedRepo.Find(cloudfeed.CloudFeed{AccountID: accountID, CloudFeedTypeID: cloudFeedTypeID})
	if err != nil {
		return cloudfeed.CloudFeed{}, err
	}

	tokenURL, refreshToken, clientID, clientSecret, err := s.cloudFeedRepo.FindOAuthInfo(accountID, cloudFeedTypeID)
	if err != nil {
		return cloudfeed.CloudFeed{}, err
	}

	if refreshToken == "" {
		return cloudfeed.CloudFeed{}, errors.New("refresh token empty")
	}

	u, err := url.Parse(tokenURL)
	if err != nil {
		return cloudfeed.CloudFeed{}, err
	}

	form := url.Values{}
	form.Add("grant_type", "refresh_token")
	form.Add("refresh_token", refreshToken)
	form.Add("client_id", clientID)
	form.Add("client_secret", clientSecret)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), strings.NewReader(form.Encode()))
	if err != nil {
		return cloudfeed.CloudFeed{}, err
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return cloudfeed.CloudFeed{}, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return cloudfeed.CloudFeed{}, errors.New("error reading response from token endpoint")
	}

	response := struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    uint   `json:"expires_in"`
		Error        string `json:"error"`
	}{}
	respBodyReader := bytes.NewReader(respBody)
	err = json.NewDecoder(respBodyReader).Decode(&response)
	if err != nil {
		return cloudfeed.CloudFeed{}, err
	}

	if resp.StatusCode != http.StatusOK {
		// Delete auth since we can not recover from "invalid_grant" error.
		if response.Error == "invalid_grant" {
			logrus.Warnln("deleting invalid cloud feed auth for accountID", accountID, "cloudFeedTypeID", cloudFeedTypeID)
			err := s.cloudFeedRepo.Delete(cloudfeed.CloudFeed{AccountID: accountID, CloudFeedTypeID: cloudFeedTypeID})
			if err != nil {
				return cloudfeed.CloudFeed{}, fmt.Errorf("error deleting invalid auth: %w", err)
			}
		}

		return cloudfeed.CloudFeed{}, fmt.Errorf("unsuccessful refresh request. request: %s", string(respBody))
	}

	cloudFeed.AccessToken = response.AccessToken
	cloudFeed.RefreshToken = response.RefreshToken

	expiryUnix := time.Now().Add(time.Second * time.Duration(response.ExpiresIn)).Unix()
	cloudFeed.Expiry = needforheat.Time(time.Unix(expiryUnix, 0))

	return s.cloudFeedRepo.Update(cloudFeed)
}

// Run this function in a goroutine to keep tokens refreshed before they expire.
// The preRenewalDuration sets the time we need to refresh the tokens in advance of theri expiry.
func (s *CloudFeedService) RefreshTokensInBackground(ctx context.Context, preRenewalDuration time.Duration) {
refreshLoop:
	for {
		accountID, cloudFeedTypeID, expiry, err := s.cloudFeedRepo.FindFirstTokenToExpire()
		if err != nil {
			logrus.Infoln("no cloud feed auths found in database. not doing anything until one is added")
			select {
			case <-s.updateChan:
				logrus.Infoln("a new cloud feed auth was added. re-checking first expiring token")
			case <-ctx.Done():
				break refreshLoop
			}
			continue
		}

		timerDuration := time.Until(time.Time(expiry)) - preRenewalDuration
		if timerDuration < 0 {
			// Wait 10 seconds to prevent a possible flood of refresh requests.
			time.Sleep(time.Second * 10)

			_, err = s.RefreshTokens(ctx, accountID, cloudFeedTypeID)
			if err != nil {
				logrus.Warningln(err)
			}
			continue
		}

		expiryTimer := time.NewTimer(timerDuration)

		logrus.Infof("waiting %s to refresh first expiring token", timerDuration.String())

		select {
		case <-expiryTimer.C:
			_, err = s.RefreshTokens(ctx, accountID, cloudFeedTypeID)
			if err != nil {
				logrus.Warningln(err)
			}
		case <-s.updateChan:
			logrus.Infoln("a new cloud feed auth was added. re-checking first expiring cloud feed auth token")
			expiryTimer.Stop()
		case <-ctx.Done():
			break refreshLoop
		}
	}
}

func exchangeAuthCode(ctx context.Context, conf *oauth2.Config, code string) (string, string, time.Time, error) {
	token, err := conf.Exchange(ctx, code, oauth2.AccessTypeOffline)
	if err != nil {
		return "", "", time.Time{}, err
	}

	return token.AccessToken, token.RefreshToken, token.Expiry, nil
}

// Run this function in a goroutine to periodically download data from the cloud feed.
// downloadStartTime is the time at which the data should be downloaded and repeated each day.
func (s *CloudFeedService) DownloadInBackground(ctx context.Context, downloadStartTime time.Time) {
	waitTime := time.Until(downloadStartTime)
	startTimer := time.NewTimer(waitTime)

	logrus.Infoln("waiting", waitTime.String(), "to start downloading data from cloud feeds")

	select {
	case <-startTimer.C:
		err := s.download(ctx)
		if err != nil {
			logrus.Errorln(err)
		}
	case <-ctx.Done():
		return
	}

	ticker := time.NewTicker(CloudFeedTypeDownloadInterval)

	for {
		waitTime = time.Until(time.Now().Add(CloudFeedTypeDownloadInterval))
		logrus.Infoln("waiting", waitTime.String(), "to download data from cloud feeds")

		select {
		case <-ticker.C:
			err := s.download(ctx)
			if err != nil {
				logrus.Errorln(err)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (s *CloudFeedService) download(ctx context.Context) error {
	cloudFeeds, err := s.cloudFeedRepo.GetAll()
	if err != nil {
		return err
	}

	logrus.Infoln("starting download of data from cloud feeds")

	for _, cfa := range cloudFeeds {
		device, err := s.cloudFeedRepo.FindDevice(cfa)
		if err != nil {
			logrus.Warningln("error finding device for cloud feed auth:", err)
			continue
		}

		latestUpload, isUpload, err := s.uploadService.GetLatestUploadTimeForDeviceWithID(device.ID)
		if err != nil && !helpers.IsMySQLRecordNotFoundError(err) {
			logrus.Warningln("error getting latest upload time for device:", err)
			continue
		}

		if latestUpload == nil || !isUpload {
			*latestUpload = NoLatestUploadTime
		}

		now := time.Now().Unix()
		unixNow := needforheat.Time(time.Unix(now, 0))

		err = s.Download(ctx, cfa, *latestUpload, unixNow)
		if err != nil {
			logrus.Warningln(err)
		}
	}

	return nil
}

// Download data from a cloud feed using the cloud feed auth and store it in the database.
// startPeriod and endPeriod are the time periods for which data should be downloaded.
func (s *CloudFeedService) Download(ctx context.Context, cfa cloudfeed.CloudFeed, startPeriod needforheat.Time, endPeriod needforheat.Time) error {
	logrus.Infoln("downloading data from cloud feed auth with accountID", cfa.AccountID, "cloudFeedTypeID", cfa.CloudFeedTypeID)

	device, err := s.cloudFeedRepo.FindDevice(cfa)
	if err != nil {
		return fmt.Errorf("error finding device for cloud feed auth: %w", err)
	}

	measurements, err := enelogic.Download(ctx, cfa.AccessToken, time.Time(startPeriod), time.Time(endPeriod))
	if err != nil {
		if err == enelogic.ErrNoData {
			logrus.Infoln("no (new) data found for cloud feed auth with accountID", cfa.AccountID, "cloudFeedTypeID", cfa.CloudFeedTypeID)
		}
		return err
	}

	if len(measurements) == 0 {
		return errors.New(fmt.Sprint("no (new) data found for cloud feed auth with accountID", cfa.AccountID, "cloudFeedTypeID", cfa.CloudFeedTypeID))
	}

	upload, err := s.uploadService.Create(device.ID, upload.Device, needforheat.Time(time.Now()), measurements)
	if err != nil {
		return errors.New(fmt.Sprint("error creating upload:", err))
	}

	if upload.Size != len(measurements) {
		return errors.New(fmt.Sprint("upload size", upload.Size, "does not match number of measurements", len(measurements)))
	}

	return nil
}
