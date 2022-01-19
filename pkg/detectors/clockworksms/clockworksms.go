package clockworksms

import (
	"context"

	// "log"
	"regexp"
	"strings"

	"net/http"

	"github.com/trufflesecurity/trufflehog/pkg/common"
	"github.com/trufflesecurity/trufflehog/pkg/detectors"
	"github.com/trufflesecurity/trufflehog/pkg/pb/detectorspb"
)

type Scanner struct{}

// Ensure the Scanner satisfies the interface at compile time
var _ detectors.Detector = (*Scanner)(nil)

var (
	client = common.SaneHttpClient()

	//Make sure that your group is surrounded in boundry characters such as below to reduce false positives
	userKeyPat = regexp.MustCompile(detectors.PrefixRegex([]string{"clockwork", "textanywhere"}) + `\b([0-9]{5})\b`)
	tokenPat   = regexp.MustCompile(detectors.PrefixRegex([]string{"clockwork", "textanywhere"}) + `\b([0-9a-zA-Z]{24})\b`)
)

// Keywords are used for efficiently pre-filtering chunks.
// Use identifiers in the secret preferably, or the provider name.
func (s Scanner) Keywords() []string {
	return []string{"clockworksms", "textanywhere"}
}

// FromData will find and optionally verify Clockworksms secrets in a given set of bytes.
func (s Scanner) FromData(ctx context.Context, verify bool, data []byte) (results []detectors.Result, err error) {
	dataStr := string(data)

	userKeyMatches := userKeyPat.FindAllStringSubmatch(dataStr, -1)
	tokenMatches := tokenPat.FindAllStringSubmatch(dataStr, -1)

	for _, match := range userKeyMatches {
		if len(match) != 2 {
			continue
		}
		resMatch := strings.TrimSpace(match[1])

		for _, tokenMatch := range tokenMatches {
			if len(tokenMatch) != 2 {
				continue
			}
			tokenRes := strings.TrimSpace(tokenMatch[1])

			s1 := detectors.Result{
				DetectorType: detectorspb.DetectorType_ClockworkSMS,
				Raw:          []byte(resMatch),
			}

			if verify {
				req, _ := http.NewRequest("GET", "https://api.textanywhere.com/API/v1.0/REST/status", nil)
				req.Header.Add("user_key", resMatch)
				req.Header.Add("access_token", tokenRes)
				res, err := client.Do(req)
				if err == nil {
					defer res.Body.Close()
					if res.StatusCode >= 200 && res.StatusCode < 300 {
						s1.Verified = true
					} else {
						//This function will check false positives for common test words, but also it will make sure the key appears 'random' enough to be a real key
						if detectors.IsKnownFalsePositive(resMatch, detectors.DefaultFalsePositives, true) {
							continue
						}
					}
				}
			}

			results = append(results, s1)
		}
	}

	return detectors.CleanResults(results), nil
}