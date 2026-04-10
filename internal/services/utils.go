package services

import (
	"crypto/rand"
	"math/big"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

// generatePassword generates a secure password meeting AWS requirements
// AWS password requirements: 8-128 characters, at least 3 of 4 character types
// (uppercase, lowercase, numbers, symbols)
func (s *AWSService) generatePassword() string {
	const (
		length    = 16
		uppercase = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
		lowercase = "abcdefghijklmnopqrstuvwxyz"
		numbers   = "0123456789"
		symbols   = "!@#$%^&*"
	)

	allChars := uppercase + lowercase + numbers + symbols
	password := make([]byte, length)

	// Ensure at least one character from each type
	charSets := []string{uppercase, lowercase, numbers, symbols}
	for i, charset := range charSets {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		password[i] = charset[n.Int64()]
	}

	// Fill the rest with random characters
	for i := len(charSets); i < length; i++ {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(allChars))))
		password[i] = allChars[n.Int64()]
	}

	// Shuffle the password to avoid predictable patterns
	for i := length - 1; i > 0; i-- {
		j, _ := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		password[i], password[j.Int64()] = password[j.Int64()], password[i]
	}

	return string(password)
}


// getRegions gets all available AWS regions (excludes opt-in regions to avoid AuthFailure)
func (s *AWSService) getRegions(sess *session.Session) ([]string, error) {
	ec2Client := ec2.New(sess)
	result, err := ec2Client.DescribeRegions(&ec2.DescribeRegionsInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("opt-in-status"),
				Values: []*string{aws.String("opt-in-not-required"), aws.String("opted-in")},
			},
		},
	})
	if err != nil {
		return nil, err
	}

	var regions []string
	for _, region := range result.Regions {
		if region.RegionName != nil {
			regions = append(regions, *region.RegionName)
		}
	}
	return regions, nil
}

// getSessionForAccountAndRegion gets a session for a specific account and region
func (s *AWSService) getSessionForAccountAndRegion(accountID, region string) (*session.Session, error) {
	sess, err := s.getSessionForAccount(accountID)
	if err != nil {
		return nil, err
	}

	// Create a copy of the session with the specific region
	return sess.Copy(&aws.Config{Region: aws.String(region)}), nil
}
