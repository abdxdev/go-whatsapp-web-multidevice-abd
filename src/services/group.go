package services

import (
	"bytes"
	"context"
	"fmt"
	"image"

	"github.com/disintegration/imaging"
	
	"github.com/aldinokemal/go-whatsapp-web-multidevice/config"
	domainGroup "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/group"
	pkgError "github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/error"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/whatsapp"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/validations"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
)

type groupService struct {
	WaCli *whatsmeow.Client
}

func NewGroupService(waCli *whatsmeow.Client) domainGroup.IGroupService {
	return &groupService{
		WaCli: waCli,
	}
}

func (service groupService) JoinGroupWithLink(ctx context.Context, request domainGroup.JoinGroupWithLinkRequest) (groupID string, err error) {
	if err = validations.ValidateJoinGroupWithLink(ctx, request); err != nil {
		return groupID, err
	}
	whatsapp.MustLogin(service.WaCli)

	jid, err := service.WaCli.JoinGroupWithLink(request.Link)
	if err != nil {
		return
	}
	return jid.String(), nil
}

func (service groupService) LeaveGroup(ctx context.Context, request domainGroup.LeaveGroupRequest) (err error) {
	if err = validations.ValidateLeaveGroup(ctx, request); err != nil {
		return err
	}

	JID, err := whatsapp.ValidateJidWithLogin(service.WaCli, request.GroupID)
	if err != nil {
		return err
	}

	return service.WaCli.LeaveGroup(JID)
}

func (service groupService) CreateGroup(ctx context.Context, request domainGroup.CreateGroupRequest) (groupID string, err error) {
	if err = validations.ValidateCreateGroup(ctx, request); err != nil {
		return groupID, err
	}
	whatsapp.MustLogin(service.WaCli)

	participantsJID, err := service.participantToJID(request.Participants)
	if err != nil {
		return
	}

	groupConfig := whatsmeow.ReqCreateGroup{
		Name:              request.Title,
		Participants:      participantsJID,
		GroupParent:       types.GroupParent{},
		GroupLinkedParent: types.GroupLinkedParent{},
	}

	groupInfo, err := service.WaCli.CreateGroup(groupConfig)
	if err != nil {
		return
	}

	return groupInfo.JID.String(), nil
}

func (service groupService) ManageParticipant(ctx context.Context, request domainGroup.ParticipantRequest) (result []domainGroup.ParticipantStatus, err error) {
	if err = validations.ValidateParticipant(ctx, request); err != nil {
		return result, err
	}
	whatsapp.MustLogin(service.WaCli)

	groupJID, err := whatsapp.ValidateJidWithLogin(service.WaCli, request.GroupID)
	if err != nil {
		return result, err
	}

	participantsJID, err := service.participantToJID(request.Participants)
	if err != nil {
		return result, err
	}

	participants, err := service.WaCli.UpdateGroupParticipants(groupJID, participantsJID, request.Action)
	if err != nil {
		return result, err
	}

	for _, participant := range participants {
		if participant.Error == 403 && participant.AddRequest != nil {
			result = append(result, domainGroup.ParticipantStatus{
				Participant: participant.JID.String(),
				Status:      "error",
				Message:     "Failed to add participant",
			})
		} else {
			result = append(result, domainGroup.ParticipantStatus{
				Participant: participant.JID.String(),
				Status:      "success",
				Message:     "Action success",
			})
		}
	}

	return result, nil
}

func (service groupService) GetGroupRequestParticipants(ctx context.Context, request domainGroup.GetGroupRequestParticipantsRequest) (result []domainGroup.GetGroupRequestParticipantsResponse, err error) {
	if err = validations.ValidateGetGroupRequestParticipants(ctx, request); err != nil {
		return result, err
	}

	groupJID, err := whatsapp.ValidateJidWithLogin(service.WaCli, request.GroupID)
	if err != nil {
		return result, err
	}

	participants, err := service.WaCli.GetGroupRequestParticipants(groupJID)
	if err != nil {
		return result, err
	}

	for _, participant := range participants {
		result = append(result, domainGroup.GetGroupRequestParticipantsResponse{
			JID:         participant.JID.String(),
			RequestedAt: participant.RequestedAt,
		})
	}

	return result, nil
}

func (service groupService) ManageGroupRequestParticipants(ctx context.Context, request domainGroup.GroupRequestParticipantsRequest) (result []domainGroup.ParticipantStatus, err error) {
	if err = validations.ValidateManageGroupRequestParticipants(ctx, request); err != nil {
		return result, err
	}

	groupJID, err := whatsapp.ValidateJidWithLogin(service.WaCli, request.GroupID)
	if err != nil {
		return result, err
	}

	participantsJID, err := service.participantToJID(request.Participants)
	if err != nil {
		return result, err
	}

	participants, err := service.WaCli.UpdateGroupRequestParticipants(groupJID, participantsJID, request.Action)
	if err != nil {
		return result, err
	}

	for _, participant := range participants {
		if participant.Error != 0 {
			result = append(result, domainGroup.ParticipantStatus{
				Participant: participant.JID.String(),
				Status:      "error",
				Message:     fmt.Sprintf("Action %s failed (code %d)", request.Action, participant.Error),
			})
		} else {
			result = append(result, domainGroup.ParticipantStatus{
				Participant: participant.JID.String(),
				Status:      "success",
				Message:     fmt.Sprintf("Action %s success", request.Action),
			})
		}
	}

	return result, nil
}

func (service groupService) ChangeGroupPhoto(ctx context.Context, request domainGroup.ChangeGroupPhotoRequest) (err error) {
	if err = validations.ValidateChangeGroupPhoto(ctx, request); err != nil {
		return err
	}
	whatsapp.MustLogin(service.WaCli)

	groupJID, err := whatsapp.ValidateJidWithLogin(service.WaCli, request.GroupID)
	if err != nil {
		return err
	}

	file, err := request.Photo.Open()
	if err != nil {
		return err
	}
	defer file.Close()

	// Read original image
	srcImage, err := imaging.Decode(file)
	if err != nil {
		return fmt.Errorf("failed to decode image: %v", err)
	}

	// Get original dimensions
	bounds := srcImage.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Calculate new dimensions for 1:1 aspect ratio
	size := width
	if height < width {
		size = height
	}
	if size > 640 {
		size = 640
	}

	// Create a square crop from the center
	left := (width - size) / 2
	top := (height - size) / 2
	croppedImage := imaging.Crop(srcImage, image.Rect(left, top, left+size, top+size))

	// Resize if needed
	if size > 640 {
		croppedImage = imaging.Resize(croppedImage, 640, 640, imaging.Lanczos)
	}

	// Convert to bytes
	var buf bytes.Buffer
	err = imaging.Encode(&buf, croppedImage, imaging.JPEG, imaging.JPEGQuality(80))
	if err != nil {
		return fmt.Errorf("failed to encode image: %v", err)
	}

	_, err = service.WaCli.SetGroupPhoto(groupJID, buf.Bytes())
	if err != nil {
		return err
	}

	return nil
}

func (service groupService) participantToJID(participants []string) ([]types.JID, error) {
	var participantsJID []types.JID
	for _, participant := range participants {
		formattedParticipant := participant + config.WhatsappTypeUser

		if !whatsapp.IsOnWhatsapp(service.WaCli, formattedParticipant) {
			return nil, pkgError.ErrUserNotRegistered
		}

		if participantJID, err := types.ParseJID(formattedParticipant); err == nil {
			participantsJID = append(participantsJID, participantJID)
		}
	}
	return participantsJID, nil
}
