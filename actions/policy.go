package actions

import (
	"encoding/json"

	"github.com/YaleSpinup/aws-go/services/iam"
	log "github.com/sirupsen/logrus"
)

func generatePolicy(actions ...string) (string, error) {
	log.Debugf("generating %v policy document", actions)

	policy := iam.PolicyDocument{
		Version: "2012-10-17",
		Statement: []iam.StatementEntry{
			{
				Effect:   "Allow",
				Action:   actions,
				Resource: []string{"*"},
			},
		},
	}

	j, err := json.Marshal(policy)
	if err != nil {
		return "", err
	}

	return string(j), nil
}
