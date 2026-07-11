package protocol

import "regexp"

var (
	versionPattern    = regexp.MustCompile(`^[0-9]+\.[0-9]+$`)
	capabilityPattern = regexp.MustCompile(`^[a-z][a-z0-9]*(?:\.[a-z][a-z0-9]*)*$`)
)

type PageMeta struct {
	ProtocolVersion      *string  `json:"protocolVersion"`
	RequiredCapabilities []string `json:"requiredCapabilities"`
}

type RendererSupport struct {
	SupportedVersions     []string `json:"supportedVersions"`
	SupportedCapabilities []string `json:"supportedCapabilities"`
}

type NegotiationResult struct {
	Accepted            bool     `json:"accepted"`
	Code                string   `json:"code"`
	PageVersion         *string  `json:"pageVersion"`
	SupportedVersions   []string `json:"supportedVersions"`
	MissingCapabilities []string `json:"missingCapabilities"`
}

func validUniqueList(values []string, pattern *regexp.Regexp, allowEmpty bool) bool {
	if !allowEmpty && len(values) == 0 {
		return false
	}
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if !pattern.MatchString(value) {
			return false
		}
		if _, exists := seen[value]; exists {
			return false
		}
		seen[value] = struct{}{}
	}
	return true
}

func negotiationResult(accepted bool, code string, pageVersion *string, supportedVersions []string) NegotiationResult {
	versions := make([]string, len(supportedVersions))
	copy(versions, supportedVersions)
	return NegotiationResult{
		Accepted:            accepted,
		Code:                code,
		PageVersion:         pageVersion,
		SupportedVersions:   versions,
		MissingCapabilities: []string{},
	}
}

func NegotiateProtocol(pageMeta PageMeta, rendererSupport RendererSupport) NegotiationResult {
	if pageMeta.ProtocolVersion == nil {
		return negotiationResult(false, "MISSING_PROTOCOL_VERSION", nil, rendererSupport.SupportedVersions)
	}
	if !versionPattern.MatchString(*pageMeta.ProtocolVersion) {
		return negotiationResult(false, "INVALID_PROTOCOL_VERSION", pageMeta.ProtocolVersion, rendererSupport.SupportedVersions)
	}
	if !validUniqueList(rendererSupport.SupportedVersions, versionPattern, false) {
		return negotiationResult(false, "INVALID_RENDERER_SUPPORT", pageMeta.ProtocolVersion, rendererSupport.SupportedVersions)
	}
	if !contains(rendererSupport.SupportedVersions, *pageMeta.ProtocolVersion) {
		return negotiationResult(false, "UNSUPPORTED_PROTOCOL_VERSION", pageMeta.ProtocolVersion, rendererSupport.SupportedVersions)
	}
	if !validUniqueList(pageMeta.RequiredCapabilities, capabilityPattern, true) {
		return negotiationResult(false, "INVALID_REQUIRED_CAPABILITIES", pageMeta.ProtocolVersion, rendererSupport.SupportedVersions)
	}
	if !validUniqueList(rendererSupport.SupportedCapabilities, capabilityPattern, true) {
		return negotiationResult(false, "INVALID_RENDERER_SUPPORT", pageMeta.ProtocolVersion, rendererSupport.SupportedVersions)
	}

	supported := make(map[string]struct{}, len(rendererSupport.SupportedCapabilities))
	for _, capability := range rendererSupport.SupportedCapabilities {
		supported[capability] = struct{}{}
	}
	missing := make([]string, 0)
	for _, capability := range pageMeta.RequiredCapabilities {
		if _, exists := supported[capability]; !exists {
			missing = append(missing, capability)
		}
	}
	if len(missing) > 0 {
		result := negotiationResult(false, "MISSING_REQUIRED_CAPABILITY", pageMeta.ProtocolVersion, rendererSupport.SupportedVersions)
		result.MissingCapabilities = missing
		return result
	}

	return negotiationResult(true, "OK", pageMeta.ProtocolVersion, rendererSupport.SupportedVersions)
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
