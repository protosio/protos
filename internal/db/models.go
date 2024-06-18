package db

import "github.com/bokwoon95/sq"

type INSTANCE struct {
	sq.TableStruct `sq:"instances"`
	VM_ID          sq.StringField `ddl:"notnull primarykey"`
	NAME           sq.StringField `ddl:"notnull index"`
	SSH_KEY_SEED   sq.StringField `ddl:"notnull"` // private SSH key stored only on the client
	PUBLIC_KEY     sq.StringField `ddl:"notnull"` // ed25519 public key
	PUBLIC_IP      sq.StringField `ddl:"notnull"` // this can be a public or private IP, depending on where the device is located
	CLOUD_TYPE     sq.StringField `ddl:"notnull"`
	CLOUD_NAME     sq.StringField `ddl:"notnull"`
	LOCATION       sq.StringField `ddl:"notnull"`
	PROTOS_VERSION sq.StringField `ddl:"notnull"`
	ARCHITECTURE   sq.StringField `ddl:"notnull"`
}

type CLOUD_PROVIDER struct {
	sq.TableStruct `sq:"cloud_providers"`
	ID             sq.StringField `ddl:"notnull primarykey"`
	NAME           sq.StringField `ddl:"notnull index"`
	TYPE           sq.StringField `ddl:"notnull"`
	AUTH           sq.JSONField   `ddl:"notnull"`
}

type SSH_KEY struct {
	sq.TableStruct `sq:"ssh_keys"`
	PRIVATE        sq.StringField `ddl:"notnull primarykey"`
	PUBLIC         sq.StringField `ddl:"notnull"`
}

type APP struct {
	sq.TableStruct `sq:"apps"`
	ID             sq.StringField  `ddl:"notnull primarykey"`
	NAME           sq.StringField  `ddl:"notnull index"`
	INSTALLER_REF  sq.StringField  `ddl:"notnull"`
	INSTANCE_NAME  sq.StringField  `ddl:"notnull"`
	DESIRED_STATUS sq.StringField  `ddl:"notnull"`
	IP             sq.StringField  `ddl:"notnull"`
	PERSISTENCE    sq.BooleanField `ddl:"notnull"`
}

type USER struct {
	sq.TableStruct `sq:"users"`
	USERNAME       sq.StringField `ddl:"notnull primarykey"`
	NAME           sq.StringField
	IS_DISABLED    sq.BooleanField `ddl:"notnull"`
}

type USER_DEVICE struct {
	sq.TableStruct `sq:"user_devices"`
	ID             sq.StringField `ddl:"notnull primarykey"`
	NAME           sq.StringField `ddl:"notnull index"`
	PUBLIC_KEY     sq.StringField `ddl:"notnull"`
	NETWORK        sq.StringField `ddl:"notnull"`
	USER_ID        sq.StringField `ddl:"notnull index"`
}
