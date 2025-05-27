package oidc

import (
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/ucerr"
)

// MaxMFAChannels is the maximum allowed number of active channels for a user
const MaxMFAChannels = 10

// MFAChannel represents an MFA channel for a user. It includes a channel type,
// channel type id, channel name, an optional challenge key, a flag indicating
// whether this is the primary channel for the user, a verified flag signifying
// whether the channel has been confirmed to be valid, and a last verified
// timestamp signifying when it was last verified.
type MFAChannel struct {
	ID            uuid.UUID      `json:"mfa_channel_id" yaml:"mfa_channel_id" validate:"notnil"`
	ChannelType   MFAChannelType `json:"mfa_channel_type" yaml:"mfa_channel_type" validate:"notempty"`
	ChannelTypeID string         `json:"mfa_channel_type_id" yaml:"mfa_channel_type_id" validate:"notempty"`
	ChannelName   string         `json:"mfa_channel_name" yaml:"mfa_channel_name" validate:"notempty"`
	ChallengeKey  string         `json:"mfa_challenge_key" yaml:"mfa_challenge_key"`
	Primary       bool           `json:"primary" yaml:"primary"`
	Verified      bool           `json:"verified" yaml:"verified"`
	LastVerified  time.Time      `json:"last_verified" yaml:"last_verified"`
}

// NewMFAChannel creates a new MFA channel. The channel is defaulted to non-verified and non-primary
func NewMFAChannel(channelType MFAChannelType, channelTypeID string, channelName string) MFAChannel {
	return MFAChannel{
		ID:            uuid.Must(uuid.NewV4()),
		ChannelType:   channelType,
		ChannelTypeID: channelTypeID,
		ChannelName:   channelName,
		Primary:       false,
		Verified:      false,
		LastVerified:  time.Time{},
	}
}

func (mfac MFAChannel) getUniqueID() string {
	return mfac.ChannelType.getChannelType().getUniqueID(mfac)
}

// GetChallengeDescription returns a challenge description appropriate for display in a UI
func (mfac MFAChannel) GetChallengeDescription(shouldMask bool, firstChallenge bool) string {
	return mfac.ChannelType.getChannelType().getChallengeDescription(mfac, shouldMask, firstChallenge)
}

// GetChannelDescription returns a channel description appropriate for display in a UI
func (mfac MFAChannel) GetChannelDescription(shouldMask bool) string {
	return mfac.ChannelType.getChannelType().getChannelDescription(mfac, shouldMask)
}

// GetRegistrationInfo returns a registration link, both in URI and QR code form, that
// can be used for registering an unverified channel with an external application
func (mfac MFAChannel) GetRegistrationInfo() (link string, qrCode string, ok bool) {
	if !mfac.Verified {
		return mfac.ChannelType.getChannelType().getRegistrationInfo(mfac)
	}

	return "", "", false
}

// GetUserDetailDescription returns a channel description appropriate for a user detail
// display in a UI
func (mfac MFAChannel) GetUserDetailDescription() string {
	return mfac.ChannelType.getChannelType().getUserDetailDescription(mfac)
}

// SetVerified marks the channel as verified with a last verified time of now
func (mfac *MFAChannel) SetVerified() {
	mfac.Verified = true
	mfac.LastVerified = time.Now().UTC()
}

func (mfac *MFAChannel) extraValidate() error {
	if err := mfac.ChannelType.getChannelType().validateChannel(mfac); err != nil {
		return ucerr.Wrap(err)
	}

	if mfac.Primary && !mfac.Verified {
		return ucerr.New("primary channel must be verified")
	}

	return nil
}

//go:generate gendbjson MFAChannel

//go:generate genvalidate MFAChannel

// MFAChannelsByID is a map from MFA channel ID to channel
type MFAChannelsByID map[uuid.UUID]MFAChannel

// MFAChannels represents a set of MFA channels, enforcing uniqueness by
// channel type and channel type id
type MFAChannels struct {
	ChannelTypes     MFAChannelTypeSet `json:"mfa_channel_types" yaml:"mfa_channel_types"`
	Channels         MFAChannelsByID   `json:"mfa_channels" yaml:"mfa_channels"`
	PrimaryChannelID uuid.UUID         `json:"primary_mfa_channel_id" yaml:"primary_mfa_channel_id"`
}

// NewMFAChannels returns an empty instance of MFAChannels
func NewMFAChannels() MFAChannels {
	return MFAChannels{
		ChannelTypes:     MFAChannelTypeSet{},
		Channels:         MFAChannelsByID{},
		PrimaryChannelID: uuid.Nil,
	}
}

// FindChannel will return the MFAChannel associate with the channel id, if it exists
func (mfac MFAChannels) FindChannel(id uuid.UUID) (MFAChannel, error) {
	c, found := mfac.Channels[id]
	if !found {
		return MFAChannel{}, ucerr.Errorf("could not find channel for id '%v'", id)
	}

	return c, nil
}

// FindPrimaryChannel will return the primary MFAChannel, if one exists and it has been verified
func (mfac MFAChannels) FindPrimaryChannel() (c MFAChannel, err error) {
	if mfac.PrimaryChannelID.IsNil() {
		return c, ucerr.New("no primary channel specified")
	}

	c, found := mfac.Channels[mfac.PrimaryChannelID]
	if !found {
		return c, ucerr.Errorf("primary channel id '%v' not found", mfac.PrimaryChannelID)
	}

	if !c.Primary {
		return c, ucerr.Errorf("primary channel id '%v' is not primary", mfac.PrimaryChannelID)
	}

	if !c.Verified {
		return c, ucerr.Errorf("primary channel id '%v' is not verified", mfac.PrimaryChannelID)
	}

	return c, nil
}

// GetVerifiedChannels returns all MFA channels of the specified channel types that have been verified,
// as well as a flag stating whether any of the channel types do not have a verified channel
func (mfac MFAChannels) GetVerifiedChannels(channelTypes MFAChannelTypeSet) (verifiedChannels MFAChannels, anyUncoveredChannelTypes bool) {
	verifiedChannels = NewMFAChannels()
	for ct := range channelTypes {
		if verifiedChannels.ChannelTypes[ct] {
			continue
		}
		verifiedChannels.ChannelTypes[ct] = true

		channelTypeCovered := false
		for _, c := range mfac.Channels {
			if c.Verified && c.ChannelType == ct {
				channelTypeCovered = true
				verifiedChannels.Channels[c.ID] = c
				if c.Primary {
					verifiedChannels.PrimaryChannelID = c.ID
				}
			}
		}
		if !channelTypeCovered {
			anyUncoveredChannelTypes = true
		}
	}

	return verifiedChannels, anyUncoveredChannelTypes
}

// HasPrimaryChannel returns true if there is a primary MFA channel
func (mfac MFAChannels) HasPrimaryChannel() bool {
	if _, err := mfac.FindPrimaryChannel(); err != nil {
		return false
	}

	return true
}

// HasVerifiedChannels returns true if there are any verified MFA channels
func (mfac MFAChannels) HasVerifiedChannels() bool {
	for _, c := range mfac.Channels {
		if c.Verified {
			return true
		}
	}

	return false
}

// AddChannel adds a new MFA channel for the specified channel type, channel type id,
// and channel name. If the resulting channels are invalid, an error is returned.
func (mfac *MFAChannels) AddChannel(channelType MFAChannelType, channelTypeID string, channelName string, setVerified bool) (channel MFAChannel, err error) {
	channel = NewMFAChannel(channelType, channelTypeID, channelName)
	channelUniqueID := channel.getUniqueID()

	for _, c := range mfac.Channels {
		if c.getUniqueID() == channelUniqueID {
			channel.ID = c.ID
			channel.LastVerified = c.LastVerified
			break
		}
	}

	if setVerified {
		channel.SetVerified()
	}

	mfac.Channels[channel.ID] = channel
	if err := mfac.Validate(); err != nil {
		return channel, ucerr.Wrap(err)
	}

	return channel, nil
}

// ClearPrimary will clear the primary channel if one exists
func (mfac *MFAChannels) ClearPrimary() {
	primaryChannel, err := mfac.FindPrimaryChannel()
	if err != nil {
		return
	}

	primaryChannel.Primary = false
	mfac.Channels[mfac.PrimaryChannelID] = primaryChannel
	mfac.PrimaryChannelID = uuid.Nil
}

// ClearUnsupportedPrimary will clear the primary channel if it does not match one of
// the specified channel types, returning true if the primary channel was cleared
func (mfac *MFAChannels) ClearUnsupportedPrimary(channelTypes MFAChannelTypeSet) bool {
	primaryChannel, err := mfac.FindPrimaryChannel()
	if err != nil {
		return false
	}

	if channelTypes[primaryChannel.ChannelType] {
		return false
	}

	mfac.ClearPrimary()
	return true
}

// ClearUnverified will remove unverified channels if any exist, return true if
// any were removed
func (mfac *MFAChannels) ClearUnverified() bool {
	anyCleared := false
	for _, c := range mfac.Channels {
		if !c.Verified {
			delete(mfac.Channels, c.ID)
			anyCleared = true
		}
	}

	return anyCleared
}

// DeleteChannel marks the specified MFA channel as unverified, returning an error if
// the channel could not be found or if it the operation could not be completed
func (mfac *MFAChannels) DeleteChannel(id uuid.UUID) error {
	channel, found := mfac.Channels[id]
	if !found {
		return ucerr.Errorf("could not find channel id '%v'", id)
	}

	if channel.Primary {
		return ucerr.Friendlyf(ucerr.Errorf("cannot delete primary channel id '%v'", id), "cannot delete primary channel")
	}

	channel.Verified = false
	mfac.Channels[id] = channel
	return nil
}

// SetPrimary will make the specified MFA channel the primary channel, clearing the previous primary
// channel in the process. An error will be returned if the specified channel could not be found or
// if it cannot be made the primary channel.
func (mfac *MFAChannels) SetPrimary(id uuid.UUID) error {
	newPrimary, found := mfac.Channels[id]
	if !found {
		return ucerr.Errorf("channel id '%v' could not be found", id)
	}

	if !newPrimary.Verified {
		return ucerr.Errorf("channel id '%v' is not verified and cannot be made primary", id)
	}

	if oldPrimary, err := mfac.FindPrimaryChannel(); err == nil {
		oldPrimary.Primary = false
		mfac.Channels[mfac.PrimaryChannelID] = oldPrimary
	}

	newPrimary.Primary = true
	mfac.Channels[id] = newPrimary
	mfac.PrimaryChannelID = id
	return nil
}

// SetVerified will mark the specified channel as being verified, with the
// last verified time set to the current time. An error will be returned if
// the channel could not be found.
func (mfac *MFAChannels) SetVerified(id uuid.UUID) error {
	channel, found := mfac.Channels[id]
	if !found {
		return ucerr.Errorf("channel id '%v' could not be found", id)
	}

	channel.SetVerified()
	mfac.Channels[id] = channel
	return nil
}

// SetVerifiedAndPrimary will mark the specified channel as being verified,
// and set the channel to be primary if a primary channel does not already
// exist
func (mfac *MFAChannels) SetVerifiedAndPrimary(id uuid.UUID) error {
	// TODO: if we start getting channel type ids from user store (e.g., email addresses,
	// phone numbers), we should keep any verified flag associated with the columns in
	// userstore in synch (e.g., if a channel type id is successfully challenged, mark it
	// as verified). We'd also want to use it as a short cut when setting up a new channel -
	// if the channel type id was already verified, we don't necessarily have to do an
	// initial challenge.
	if err := mfac.SetVerified(id); err != nil {
		return ucerr.Wrap(err)
	}

	if !mfac.HasPrimaryChannel() {
		if err := mfac.SetPrimary(id); err != nil {
			return ucerr.Wrap(err)
		}
	}

	return nil
}

// UpdateChannel will update the specified channel, returning an error if the
// channel ID could not be found or if the resulting channels are now invalid
func (mfac *MFAChannels) UpdateChannel(channel MFAChannel) error {
	if _, found := mfac.Channels[channel.ID]; !found {
		return ucerr.Errorf("channel id '%v' could not be found", channel.ID)
	}

	mfac.Channels[channel.ID] = channel
	return ucerr.Wrap(mfac.Validate())
}

// Validate implements the Validatable interface
func (mfac MFAChannels) Validate() error {
	for ct := range mfac.ChannelTypes {
		if err := ct.Validate(); err != nil {
			return ucerr.Wrap(err)
		}
	}

	if len(mfac.Channels) == 0 {
		return nil
	}

	totalVerifiedNonRecoveryCodeChannels := 0

	uniqueIDs := map[string]bool{}
	for _, c := range mfac.Channels {
		if err := c.Validate(); err != nil {
			return ucerr.Wrap(err)
		}

		if len(mfac.ChannelTypes) > 0 && !mfac.ChannelTypes[c.ChannelType] {
			return ucerr.Errorf("unsupported channel type '%v' for channel ID '%v'", c.ChannelType, c.ID)
		}

		uniqueID := c.getUniqueID()
		if uniqueIDs[uniqueID] {
			return ucerr.Friendlyf(ucerr.Errorf("duplicate channel exists with unique ID '%s'", uniqueID), "this method already exists")
		}
		uniqueIDs[uniqueID] = true

		if c.Primary && c.ID != mfac.PrimaryChannelID {
			return ucerr.Errorf("channel '%v' is primary but does not match primary channel id '%v'", c.ID, mfac.PrimaryChannelID)
		}

		if c.Verified && !c.ChannelType.IsRecoveryCode() {
			totalVerifiedNonRecoveryCodeChannels++
		}
	}

	if totalVerifiedNonRecoveryCodeChannels > MaxMFAChannels {
		return ucerr.Friendlyf(nil, "cannot have more than %d active channels", MaxMFAChannels)
	}

	return nil
}

//go:generate gendbjson MFAChannels
