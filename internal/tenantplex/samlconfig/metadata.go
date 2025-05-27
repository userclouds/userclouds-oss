package samlconfig

import (
	"encoding/xml"
	"time"

	"github.com/beevik/etree"
	"github.com/crewjam/saml"

	"userclouds.com/infra/ucerr"
)

// NB: this whole file is pulled in largely because we need to be able to add
// JSON tags to the saml.EntityDescriptor struct. Tried to fork as little as possible
// to get started.

// HTTPPostBinding is the official URN for the HTTP-POST binding (transport)
const HTTPPostBinding = "urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST"

// HTTPRedirectBinding is the official URN for the HTTP-Redirect binding (transport)
const HTTPRedirectBinding = "urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect"

// HTTPArtifactBinding is the official URN for the HTTP-Artifact binding (transport)
const HTTPArtifactBinding = "urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Artifact"

// SOAPBinding is the official URN for the SOAP binding (transport)
const SOAPBinding = "urn:oasis:names:tc:SAML:2.0:bindings:SOAP"

// EntitiesDescriptor represents the SAML object of the same name.
//
// See http://docs.oasis-open.org/security/saml/v2.0/saml-metadata-2.0-os.pdf §2.3.1
type EntitiesDescriptor struct {
	XMLName             xml.Name             `json:"xml_name" xml:"urn:oasis:names:tc:SAML:2.0:metadata EntitiesDescriptor"`
	ID                  *string              `json:"id" xml:",attr,omitempty"`
	ValidUntil          *time.Time           `json:"valid_until" xml:"validUntil,attr,omitempty"`
	CacheDuration       *time.Duration       `json:"cache_duration" xml:"cacheDuration,attr,omitempty"`
	Name                *string              `json:"name" xml:",attr,omitempty"`
	Signature           *etree.Element       `json:"-"`
	EntitiesDescriptors []EntitiesDescriptor `json:"entities_descriptors" xml:"urn:oasis:names:tc:SAML:2.0:metadata EntitiesDescriptor"`
	EntityDescriptors   []EntityDescriptor   `json:"entity_descriptors" xml:"urn:oasis:names:tc:SAML:2.0:metadata EntityDescriptor"`
}

// EntityDescriptor represents the SAML EntityDescriptor object.
//
// See http://docs.oasis-open.org/security/saml/v2.0/saml-metadata-2.0-os.pdf §2.3.2
type EntityDescriptor struct {
	XMLName                       xml.Name                       `json:"xml_name" xml:"urn:oasis:names:tc:SAML:2.0:metadata EntityDescriptor"`
	EntityID                      string                         `json:"entity_id" xml:"entityID,attr"`
	ID                            string                         `json:"id" xml:",attr,omitempty"`
	ValidUntil                    time.Time                      `json:"valid_until" xml:"validUntil,attr,omitempty"`
	CacheDuration                 time.Duration                  `json:"cache_duration" xml:"cacheDuration,attr,omitempty"`
	Signature                     *etree.Element                 `json:"-"`
	RoleDescriptors               []RoleDescriptor               `json:"role_descriptors" xml:"RoleDescriptor"`
	IDPSSODescriptors             []IDPSSODescriptor             `json:"idp_sso_descriptors" xml:"IDPSSODescriptor"`
	SPSSODescriptors              []SPSSODescriptor              `json:"sp_sso_descriptors" xml:"SPSSODescriptor"`
	AuthnAuthorityDescriptors     []AuthnAuthorityDescriptor     `json:"authn_authority_descriptors" xml:"AuthnAuthorityDescriptor"`
	AttributeAuthorityDescriptors []AttributeAuthorityDescriptor `json:"attribute_authority_descriptors" xml:"AttributeAuthorityDescriptor"`
	PDPDescriptors                []PDPDescriptor                `json:"pdp_descriptors" xml:"PDPDescriptor"`
	AffiliationDescriptor         *AffiliationDescriptor         `json:"affiliation_descriptor,omitempty"`
	Organization                  *Organization                  `json:"organization,omitempty"`
	ContactPerson                 *ContactPerson                 `json:"contact_person,omitempty"`
	AdditionalMetadataLocations   []string                       `json:"additional_metadata_locations" xml:"AdditionalMetadataLocation"`
}

// MarshalXML implements xml.Marshaler
func (m EntityDescriptor) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
	type Alias EntityDescriptor
	aux := &struct {
		ValidUntil    saml.RelaxedTime `xml:"validUntil,attr,omitempty"`
		CacheDuration saml.Duration    `xml:"cacheDuration,attr,omitempty"`
		*Alias
	}{
		ValidUntil:    saml.RelaxedTime(m.ValidUntil),
		CacheDuration: saml.Duration(m.CacheDuration),
		Alias:         (*Alias)(&m),
	}
	return ucerr.Wrap(e.Encode(aux))
}

// UnmarshalXML implements xml.Unmarshaler
func (m *EntityDescriptor) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	type Alias EntityDescriptor
	aux := &struct {
		ValidUntil    saml.RelaxedTime `xml:"validUntil,attr,omitempty"`
		CacheDuration saml.Duration    `xml:"cacheDuration,attr,omitempty"`
		*Alias
	}{
		Alias: (*Alias)(m),
	}
	if err := d.DecodeElement(aux, &start); err != nil {
		return ucerr.Wrap(err)
	}
	m.ValidUntil = time.Time(aux.ValidUntil)
	m.CacheDuration = time.Duration(aux.CacheDuration)
	return nil
}

// Organization represents the SAML Organization object.
//
// See http://docs.oasis-open.org/security/saml/v2.0/saml-metadata-2.0-os.pdf §2.3.2.1
type Organization struct {
	OrganizationNames        []LocalizedName `json:"organization_names" xml:"OrganizationName"`
	OrganizationDisplayNames []LocalizedName `json:"organization_display_names" xml:"OrganizationDisplayName"`
	OrganizationURLs         []LocalizedURI  `json:"organization_urls" xml:"OrganizationURL"`
}

// LocalizedName represents the SAML type localizedNameType.
//
// See http://docs.oasis-open.org/security/saml/v2.0/saml-metadata-2.0-os.pdf §2.2.4
type LocalizedName struct {
	Lang  string `json:"lang" xml:"http://www.w3.org/XML/1998/namespace lang,attr"`
	Value string `json:"value" xml:",chardata"`
}

// LocalizedURI represents the SAML type localizedURIType.
//
// See http://docs.oasis-open.org/security/saml/v2.0/saml-metadata-2.0-os.pdf §2.2.5
type LocalizedURI struct {
	Lang  string `json:"lang" xml:"http://www.w3.org/XML/1998/namespace lang,attr"`
	Value string `json:"value" xml:",chardata"`
}

// ContactPerson represents the SAML element ContactPerson.
//
// See http://docs.oasis-open.org/security/saml/v2.0/saml-metadata-2.0-os.pdf §2.3.2.2
type ContactPerson struct {
	ContactType      string   `json:"contact_type" xml:"contactType,attr"`
	Company          string   `json:"company"`
	GivenName        string   `json:"given_name"`
	SurName          string   `json:"surname"`
	EmailAddresses   []string `json:"email_addresses" xml:"EmailAddress"`
	TelephoneNumbers []string `json:"telephone_numbers" xml:"TelephoneNumber"`
}

// RoleDescriptor represents the SAML element RoleDescriptor.
//
// See http://docs.oasis-open.org/security/saml/v2.0/saml-metadata-2.0-os.pdf §2.4.1
type RoleDescriptor struct {
	ID                         string          `json:"id" xml:",attr,omitempty"`
	ValidUntil                 *time.Time      `json:"valid_until" xml:"validUntil,attr,omitempty"`
	CacheDuration              time.Duration   `json:"cache_duration" xml:"cacheDuration,attr,omitempty"`
	ProtocolSupportEnumeration string          `json:"protocol_support_enumeration" xml:"protocolSupportEnumeration,attr"`
	ErrorURL                   string          `json:"error_url" xml:"errorURL,attr,omitempty"`
	Signature                  *etree.Element  `json:"-"`
	KeyDescriptors             []KeyDescriptor `json:"key_descriptors" xml:"KeyDescriptor,omitempty"`
	Organization               *Organization   `json:"organization" xml:"Organization,omitempty"`
	ContactPeople              []ContactPerson `json:"contact_people" xml:"ContactPerson,omitempty"`
}

// KeyDescriptor represents the XMLSEC object of the same name
type KeyDescriptor struct {
	Use               string             `json:"use" xml:"use,attr"`
	KeyInfo           KeyInfo            `json:"key_info" xml:"http://www.w3.org/2000/09/xmldsig# KeyInfo"`
	EncryptionMethods []EncryptionMethod `json:"encryption_methods" xml:"EncryptionMethod"`
}

// EncryptionMethod represents the XMLSEC object of the same name
type EncryptionMethod struct {
	Algorithm string `json:"algorithm" xml:"Algorithm,attr"`
}

// KeyInfo represents the XMLSEC object of the same name
type KeyInfo struct {
	XMLName  xml.Name `json:"xml_name" xml:"http://www.w3.org/2000/09/xmldsig# KeyInfo"`
	X509Data X509Data `json:"x509_data" xml:"X509Data"`
}

// X509Data represents the XMLSEC object of the same name
type X509Data struct {
	XMLName          xml.Name          `json:"xml_name" xml:"http://www.w3.org/2000/09/xmldsig# X509Data"`
	X509Certificates []X509Certificate `json:"x509_certificates" xml:"X509Certificate"`
}

// X509Certificate represents the XMLSEC object of the same name
type X509Certificate struct {
	XMLName xml.Name `json:"xml_name" xml:"http://www.w3.org/2000/09/xmldsig# X509Certificate"`
	Data    string   `json:"data" xml:",chardata"`
}

// Endpoint represents the SAML EndpointType object.
//
// See http://docs.oasis-open.org/security/saml/v2.0/saml-metadata-2.0-os.pdf §2.2.2
type Endpoint struct {
	Binding          string `json:"binding" xml:"Binding,attr"`
	Location         string `json:"location" xml:"Location,attr"`
	ResponseLocation string `json:"response_location" xml:"ResponseLocation,attr,omitempty"`
}

// IndexedEndpoint represents the SAML IndexedEndpointType object.
//
// See http://docs.oasis-open.org/security/saml/v2.0/saml-metadata-2.0-os.pdf §2.2.3
type IndexedEndpoint struct {
	Binding          string  `json:"binding" xml:"Binding,attr"`
	Location         string  `json:"location" xml:"Location,attr"`
	ResponseLocation *string `json:"response_location" xml:"ResponseLocation,attr,omitempty"`
	Index            int     `json:"index" xml:"index,attr"`
	IsDefault        *bool   `json:"is_default,omitempty" xml:"isDefault,attr"`
}

// SSODescriptor represents the SAML complex type SSODescriptor
//
// See http://docs.oasis-open.org/security/saml/v2.0/saml-metadata-2.0-os.pdf §2.4.2
type SSODescriptor struct {
	RoleDescriptor
	ArtifactResolutionServices []IndexedEndpoint `json:"artifact_resolution_services" xml:"ArtifactResolutionService"`
	SingleLogoutServices       []Endpoint        `json:"single_logout_services" xml:"SingleLogoutService"`
	ManageNameIDServices       []Endpoint        `json:"manage_name_id_services" xml:"ManageNameIDService"`
	NameIDFormats              []NameIDFormat    `json:"name_id_formats" xml:"NameIDFormat"`
}

// IDPSSODescriptor represents the SAML IDPSSODescriptorType object.
//
// See http://docs.oasis-open.org/security/saml/v2.0/saml-metadata-2.0-os.pdf §2.4.3
type IDPSSODescriptor struct {
	XMLName xml.Name `json:"xml_name" xml:"urn:oasis:names:tc:SAML:2.0:metadata IDPSSODescriptor"`
	SSODescriptor
	WantAuthnRequestsSigned *bool `json:"want_authn_requests_signed,omitempty" xml:",attr"`

	SingleSignOnServices       []Endpoint       `json:"single_sign_on_services" xml:"SingleSignOnService"`
	ArtifactResolutionServices []Endpoint       `json:"artifact_resolution_servies" xml:"ArtifactResolutionService"`
	NameIDMappingServices      []Endpoint       `json:"name_id_mapping_services" xml:"NameIDMappingService"`
	AssertionIDRequestServices []Endpoint       `json:"assertion_id_request_services" xml:"AssertionIDRequestService"`
	AttributeProfiles          []string         `json:"attribute_profiles" xml:"AttributeProfile"`
	Attributes                 []saml.Attribute `json:"attributes" xml:"Attribute"`
}

// SPSSODescriptor represents the SAML SPSSODescriptorType object.
//
// See http://docs.oasis-open.org/security/saml/v2.0/saml-metadata-2.0-os.pdf §2.4.2
type SPSSODescriptor struct {
	XMLName xml.Name `json:"xml_name" xml:"urn:oasis:names:tc:SAML:2.0:metadata SPSSODescriptor"`
	SSODescriptor
	AuthnRequestsSigned        *bool                       `json:"authn_requests_signed,omitempty" xml:",attr"`
	WantAssertionsSigned       *bool                       `json:"want_assertions_signed,omitempty" xml:",attr"`
	AssertionConsumerServices  []IndexedEndpoint           `json:"assertion_consumer_services" xml:"AssertionConsumerService"`
	AttributeConsumingServices []AttributeConsumingService `json:"attribute_consuming_services" xml:"AttributeConsumingService"`
}

// AttributeConsumingService represents the SAML AttributeConsumingService object.
//
// See http://docs.oasis-open.org/security/saml/v2.0/saml-metadata-2.0-os.pdf §2.4.4.1
type AttributeConsumingService struct {
	Index               int                  `json:"index" xml:"index,attr"`
	IsDefault           *bool                `json:"is_default,omitempty" xml:"isDefault,attr"`
	ServiceNames        []LocalizedName      `json:"service_names" xml:"ServiceName"`
	ServiceDescriptions []LocalizedName      `json:"service_descriptions" xml:"ServiceDescription"`
	RequestedAttributes []RequestedAttribute `json:"requested_attributes" xml:"RequestedAttribute"`
}

// RequestedAttribute represents the SAML RequestedAttribute object.
//
// See http://docs.oasis-open.org/security/saml/v2.0/saml-metadata-2.0-os.pdf §2.4.4.2
type RequestedAttribute struct {
	saml.Attribute
	IsRequired *bool `json:"is_required,omitempty" xml:"isRequired,attr"`
}

// AuthnAuthorityDescriptor represents the SAML AuthnAuthorityDescriptor object.
//
// See http://docs.oasis-open.org/security/saml/v2.0/saml-metadata-2.0-os.pdf §2.4.5
type AuthnAuthorityDescriptor struct {
	RoleDescriptor
	AuthnQueryServices         []Endpoint     `json:"authn_query_services" xml:"AuthnQueryService"`
	AssertionIDRequestServices []Endpoint     `json:"assertion_id_request_services" xml:"AssertionIDRequestService"`
	NameIDFormats              []NameIDFormat `json:"name_id_formats" xml:"NameIDFormat"`
}

// PDPDescriptor represents the SAML PDPDescriptor object.
//
// See http://docs.oasis-open.org/security/saml/v2.0/saml-metadata-2.0-os.pdf §2.4.6
type PDPDescriptor struct {
	RoleDescriptor
	AuthzServices              []Endpoint     `json:"authz_services" xml:"AuthzService"`
	AssertionIDRequestServices []Endpoint     `json:"assertion_id_request_services" xml:"AssertionIDRequestService"`
	NameIDFormats              []NameIDFormat `json:"name_id_formats" xml:"NameIDFormat"`
}

// AttributeAuthorityDescriptor represents the SAML AttributeAuthorityDescriptor object.
//
// See http://docs.oasis-open.org/security/saml/v2.0/saml-metadata-2.0-os.pdf §2.4.7
type AttributeAuthorityDescriptor struct {
	RoleDescriptor
	AttributeServices          []Endpoint       `json:"attribute_services" xml:"AttributeService"`
	AssertionIDRequestServices []Endpoint       `json:"assertion_id_request_services" xml:"AssertionIDRequestService"`
	NameIDFormats              []NameIDFormat   `json:"name_id_formats" xml:"NameIDFormat"`
	AttributeProfiles          []string         `json:"attribute_profiles" xml:"AttributeProfile"`
	Attributes                 []saml.Attribute `json:"attributes" xml:"Attribute"`
}

// AffiliationDescriptor represents the SAML AffiliationDescriptor object.
//
// See http://docs.oasis-open.org/security/saml/v2.0/saml-metadata-2.0-os.pdf §2.5
type AffiliationDescriptor struct {
	AffiliationOwnerID string          `json:"affiliation_owner_id" xml:"affiliationOwnerID,attr"`
	ID                 string          `json:"id" xml:",attr"`
	ValidUntil         time.Time       `json:"valid_until" xml:"validUntil,attr,omitempty"`
	CacheDuration      time.Duration   `json:"cache_duration" xml:"cacheDuration,attr"`
	Signature          *etree.Element  `json:"-"`
	AffiliateMembers   []string        `json:"affiliate_members" xml:"AffiliateMember"`
	KeyDescriptors     []KeyDescriptor `json:"key_descriptors" xml:"KeyDescriptor"`
}

// NameIDFormat is the format of the id
type NameIDFormat string

// Element returns an XML element representation of n.
func (n NameIDFormat) Element() *etree.Element {
	el := etree.NewElement("")
	el.SetText(string(n))
	return el
}
