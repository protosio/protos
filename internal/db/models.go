package db

import "github.com/bokwoon95/sq"

type INSTANCE struct {
	sq.TableStruct `sq:"instances"`
	VM_ID          sq.StringField
	NAME           sq.StringField
	SSH_KEY_SEED   sq.StringField // private SSH key stored only on the client
	PUBLIC_KEY     sq.StringField // ed25519 public key
	PUBLIC_IP      sq.StringField // this can be a public or private IP, depending on where the device is located
	INTERNAL_IP    sq.StringField // this is the wireguard IP
	CLOUD_TYPE     sq.StringField
	CLOUD_NAME     sq.StringField
	LOCATION       sq.StringField
	NETWORK        sq.StringField
	PROTOS_VERSION sq.StringField
	ARCHITECTURE   sq.StringField
}

type CLOUD_PROVIDER struct {
	sq.TableStruct `sq:"cloud_providers"`
	NAME           sq.StringField
	TYPE           sq.StringField
	AUTH           sq.JSONField
}

type SSH_KEY struct {
	sq.TableStruct `sq:"ssh_keys"`
	PRIVATE        sq.StringField
	PUBLIC         sq.StringField
}

type APP struct {
	sq.TableStruct `sq:"apps"`
	NAME           sq.StringField
	ID             sq.StringField
	INSTALLER_REF  sq.StringField
	INSTANCE_NAME  sq.StringField
	DESIRED_STATUS sq.StringField
	IP             sq.StringField
	PERSISTENCE    sq.BooleanField
}

type USER struct {
	sq.TableStruct `sq:"users"`
	USERNAME       sq.StringField
	NAME           sq.StringField
	IS_DISABLED    sq.BooleanField
	DEVICES        sq.ArrayField
}

type USER_DEVICE struct {
	sq.TableStruct `sq:"user_devices"`
	ID             sq.StringField
	NAME           sq.StringField
	PUBLIC_KEY     sq.StringField
	NETWORK        sq.StringField
}
