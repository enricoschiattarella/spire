package idutil

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

type idType int

const (
	anyId idType = iota
	trustDomainId
	workloadId
	agentId
	serverId
)

// ValidateSpiffeID validates the SPIFFE ID according to the SPIFFE
// specification. The validation mode controls the type of validation.
func ValidateSpiffeID(spiffeID string, mode ValidationMode) error {
	_, err := ParseSpiffeID(spiffeID, mode)
	return err
}

// ValidateSpiffeIDURL validates the SPIFFE ID according to the SPIFFE
// specification, namely:
// - spiffe id is not empty
// - spiffe id is a valid url
// - scheme is 'spiffe'
// - user info is not allowed
// - host is not empty
// - port is not allowed
// - query values are not allowed
// - fragment is not allowed
// - path does not start with '/spire' since it is reserved for agent, server, etc.
// In addition, the validation mode is used to control what kind of SPIFFE ID
// is expected.
// For more information:
// [https://github.com/spiffe/spiffe/blob/master/standards/SPIFFE-ID.md]
func ValidateSpiffeIDURL(id *url.URL, mode ValidationMode) error {
	options := mode.validationOptions()

	validationError := func(format string, args ...interface{}) error {
		var kind string
		switch options.idType {
		case trustDomainId:
			kind = "trust domain "
		case workloadId:
			kind = "workload "
		case serverId:
			kind = "server "
		case agentId:
			kind = "agent "
		}
		return fmt.Errorf("%q is not a valid %sSPIFFE ID: "+format,
			append([]interface{}{id.String(), kind}, args...)...)
	}

	if id == nil || *id == (url.URL{}) {
		return validationError("SPIFFE ID is empty")
	}

	// General validation
	switch {
	case id.Scheme != "spiffe":
		return validationError("invalid scheme")
	case id.User != nil:
		return validationError("user info is not allowed")
	case id.Host == "":
		return validationError("trust domain is empty")
	case id.Port() != "":
		return validationError("port is not allowed")
	case id.Fragment != "":
		return validationError("fragment is not allowed")
	case id.RawQuery != "":
		return validationError("query is not allowed")
	}

	// trust domain validation
	if options.trustDomainRequired {
		if options.trustDomain == "" {
			return errors.New("trust domain to validate against cannot be empty")
		}
		if id.Host != options.trustDomain {
			return fmt.Errorf("%q does not belong to trust domain %q", id, options.trustDomain)
		}
	}

	// id type validation
	switch options.idType {
	case anyId:
	case trustDomainId:
		if id.Path != "" {
			return validationError("path is not empty")
		}
	case workloadId:
		if id.Path == "" {
			return validationError("path is empty")
		}
		if isReservedPath(id.Path) {
			return validationError(`invalid path: "/spire/*" namespace is reserved`)
		}
	case serverId:
		if id.Path == "" {
			return validationError("path is empty")
		}
		if !isServerPath(id.Path) {
			return validationError(`invalid path: expecting "/spire/server"`)
		}
	case agentId:
		if id.Path == "" {
			return validationError("path is empty")
		}
		if !isAgentPath(id.Path) {
			return validationError(`invalid path: expecting "/spire/agent/*"`)
		}
	default:
		return validationError("internal error: unhandled id type %v", options.idType)
	}

	return nil
}

func isReservedPath(path string) bool {
	return path == "/spire" || strings.HasPrefix(path, "/spire/")
}

func isServerPath(path string) bool {
	return path == "/spire/server"
}

func isAgentPath(path string) bool {
	return strings.HasPrefix(path, "/spire/agent/")
}

// ParseSpiffeID parses the SPIFFE ID and makes sure it is valid according to
// the specified validation mode.
func ParseSpiffeID(spiffeID string, mode ValidationMode) (*url.URL, error) {
	u, err := url.Parse(spiffeID)
	if err != nil {
		return nil, fmt.Errorf("could not parse SPIFFE ID: %v", err)
	}

	if err := ValidateSpiffeIDURL(u, mode); err != nil {
		return nil, err
	}
	return u, nil
}

type ValidationMode interface {
	validationOptions() validationOptions
}

type validationOptions struct {
	trustDomain         string
	trustDomainRequired bool
	idType              idType
}

type validationMode struct {
	options validationOptions
}

func (m validationMode) validationOptions() validationOptions {
	return m.options
}

// Allows any well-formed SPIFFE ID
func AllowAny() ValidationMode {
	return validationMode{}
}

// Allows any well-formed SPIFFE ID either for, or belonging to, a specific trust domain.
func AllowAnyInTrustDomain(trustDomain string) ValidationMode {
	return validationMode{
		options: validationOptions{
			trustDomain:         trustDomain,
			trustDomainRequired: true,
		},
	}
}

// Allows a well-formed SPIFFE ID for the specific trust domain.
func AllowTrustDomain(trustDomain string) ValidationMode {
	return validationMode{
		options: validationOptions{
			trustDomain:         trustDomain,
			trustDomainRequired: true,
			idType:              trustDomainId,
		},
	}
}

// Allows a well-formed SPIFFE ID for a workload belonging to a specific trust domain.
func AllowTrustDomainWorkload(trustDomain string) ValidationMode {
	return validationMode{
		options: validationOptions{
			trustDomain:         trustDomain,
			trustDomainRequired: true,
			idType:              workloadId,
		},
	}
}

func AllowTrustDomainServer(trustDomain string) ValidationMode {
	return validationMode{
		options: validationOptions{
			trustDomain:         trustDomain,
			trustDomainRequired: true,
			idType:              serverId,
		},
	}
}

func AllowTrustDomainAgent(trustDomain string) ValidationMode {
	return validationMode{
		options: validationOptions{
			trustDomain:         trustDomain,
			trustDomainRequired: true,
			idType:              agentId,
		},
	}
}

// Allows a well-formed SPIFFE ID for any trust domain.
func AllowAnyTrustDomain() ValidationMode {
	return validationMode{
		options: validationOptions{
			idType: trustDomainId,
		},
	}
}

// Allows a well-formed SPIFFE ID for a workload belonging to any trust domain.
func AllowAnyTrustDomainWorkload() ValidationMode {
	return validationMode{
		options: validationOptions{
			idType: workloadId,
		},
	}
}

func AllowAnyTrustDomainServer() ValidationMode {
	return validationMode{
		options: validationOptions{
			idType: serverId,
		},
	}
}

func AllowAnyTrustDomainAgent() ValidationMode {
	return validationMode{
		options: validationOptions{
			idType: agentId,
		},
	}
}
