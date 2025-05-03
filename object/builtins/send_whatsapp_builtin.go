package builtins

import (
	"zumbra/object"

	twilio "github.com/twilio/twilio-go"
	openapi "github.com/twilio/twilio-go/rest/api/v2010"
)

func SendWhatsappBuiltin() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {

			if len(args) != 1 {
				return NewError("wrong number of arguments. got=%d, want=1", len(args))
			}

			if args[0].Type() != object.DICT_OBJ {
				return NewError("argument to `sendWhatsapp` must be DICT, with the fields {sid, auth, senderPhone, receiverPhone, message}, got %s", args[0].Type())
			}

			dict := args[0].(*object.Dict)
			keySid := &object.String{Value: "sid"}
			keyAuth := &object.String{Value: "auth"}
			KeySenderPhone := &object.String{Value: "senderPhone"}
			keyReceiverPhone := &object.String{Value: "receiverPhone"}
			keyMessage := &object.String{Value: "message"}

			pairSid, okSid := dict.Pairs[keySid.DictKey()]
			pairAuth, okAuth := dict.Pairs[keyAuth.DictKey()]
			pairSenderPhone, okSenderPhone := dict.Pairs[KeySenderPhone.DictKey()]
			pairReceiverPhone, okReceiverPhone := dict.Pairs[keyReceiverPhone.DictKey()]
			pairMessage, okMessage := dict.Pairs[keyMessage.DictKey()]

			if !okSid || !okAuth || !okSenderPhone || !okReceiverPhone || !okMessage {
				return NewError("missing 'sid' or 'auth' or 'senderPhone' or 'receiverPhone' or 'message'")
			}

			sidStr, ok := pairSid.Value.(*object.String)
			authStr, ok := pairAuth.Value.(*object.String)
			senderPhoneStr, ok := pairSenderPhone.Value.(*object.String)
			receiverPhoneStr, ok := pairReceiverPhone.Value.(*object.String)
			messageStr, ok := pairMessage.Value.(*object.String)

			if !ok {
				return NewError("All fields must be STRING")
			}

			client := twilio.NewRestClientWithParams(twilio.ClientParams{
				Username: sidStr.Value,
				Password: authStr.Value,
			})

			params := &openapi.CreateMessageParams{}
			params.SetTo("whatsapp:+" + receiverPhoneStr.Value)
			params.SetFrom("whatsapp:+" + senderPhoneStr.Value)
			params.SetBody(messageStr.Value)

			_, err := client.Api.CreateMessage(params)
			if err != nil {
				return NewError("%s", err.Error())
			}
			return NewString("Message sent successfully!")
		},
	}
}
