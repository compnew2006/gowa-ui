package whatsmeow

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"go.mau.fi/whatsmeow/proto/waCompanionReg"
	"go.mau.fi/whatsmeow/proto/waWa6"
	waStore "go.mau.fi/whatsmeow/store"
	"google.golang.org/protobuf/proto"
)

const maxLinkedDeviceNameRunes = 64

func buildLinkedDeviceName(identityPrefix, instanceName string, instanceID uuid.UUID) string {
	prefix := strings.TrimSpace(identityPrefix)
	name := strings.TrimSpace(instanceName)
	if name == "" {
		if prefix != "" {
			name = instanceID.String()[:8]
		} else {
			name = fmt.Sprintf("Whatomate %s", instanceID.String()[:8])
		}
	}
	if prefix != "" {
		if strings.HasSuffix(prefix, "+") {
			name = prefix + name
		} else {
			name = prefix + "+" + name
		}
	}
	runes := []rune(name)
	if len(runes) > maxLinkedDeviceNameRunes {
		return string(runes[:maxLinkedDeviceNameRunes])
	}
	return name
}

func applyLinkedDeviceName(payload *waWa6.ClientPayload, linkedDeviceName string) {
	if payload == nil {
		return
	}
	name := strings.TrimSpace(linkedDeviceName)
	if name == "" {
		return
	}

	if payload.UserAgent != nil {
		payload.UserAgent.Device = proto.String(name)
	}

	if payload.DevicePairingData == nil {
		return
	}

	var deviceProps *waCompanionReg.DeviceProps
	if raw := payload.DevicePairingData.GetDeviceProps(); len(raw) > 0 {
		parsed := &waCompanionReg.DeviceProps{}
		if err := proto.Unmarshal(raw, parsed); err == nil {
			deviceProps = parsed
		}
	}

	if deviceProps == nil {
		if waStore.DeviceProps != nil {
			deviceProps = proto.Clone(waStore.DeviceProps).(*waCompanionReg.DeviceProps)
		} else {
			deviceProps = &waCompanionReg.DeviceProps{}
		}
	}

	deviceProps.Os = proto.String(name)
	if encoded, err := proto.Marshal(deviceProps); err == nil {
		payload.DevicePairingData.DeviceProps = encoded
	}
}
