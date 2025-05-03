package builtins

import (
	"net/smtp"
	"strings"
	"zumbra/object"
)

type MessageBody struct {
	sender  string
	to      string
	subject string
	body    string
}

func SendEmailBuiltin() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {

			if len(args) != 1 {
				return NewError("wrong number of arguments. got=%d, want=1", len(args))
			}

			if args[0].Type() != object.DICT_OBJ {
				return NewError("argument to `sendEmail` must be DICT, with the fields {subject, body, sender, to}, got %s", args[0].Type())
			}

			dict := args[0].(*object.Dict)
			keySubject := &object.String{Value: "subject"}
			keyBody := &object.String{Value: "body"}
			keySender := &object.String{Value: "sender"}
			keyTo := &object.String{Value: "to"}
			keyPass := &object.String{Value: "app_password"}

			pairSubject, okSubject := dict.Pairs[keySubject.DictKey()]
			pairBody, okBody := dict.Pairs[keyBody.DictKey()]
			pairSender, okSender := dict.Pairs[keySender.DictKey()]
			pairTo, okTo := dict.Pairs[keyTo.DictKey()]
			pairPass, okPass := dict.Pairs[keyPass.DictKey()]

			if !okSubject || !okBody || !okSender || !okTo || !okPass {
				return NewError("missing 'subject' or 'body'")
			}

			subjectStr, ok := pairSubject.Value.(*object.String)
			bodyStr, ok2 := pairBody.Value.(*object.String)
			senderStr, ok2 := pairSender.Value.(*object.String)
			toStr, ok2 := pairTo.Value.(*object.String)
			passStr, ok2 := pairPass.Value.(*object.String)

			if !ok || !ok2 {
				return NewError("'subject' or 'body' must be strings")
			}

			return NewString(sendEmail(toStr.Value, senderStr.Value, subjectStr.Value, bodyStr.Value, strings.ReplaceAll(passStr.Value, " ", "")).Value)

		},
	}
}

func sendEmail(To string, Sender string, Subject string, Body string, Password string) object.String {
	to := []string{To}
	space := "\r\n"
	subject := "Subject: " + Subject
	body := Body

	msg := []byte(subject + space + space + body + space)

	auth := smtp.PlainAuth("", Sender, Password, "smtp.gmail.com")

	err := smtp.SendMail("smtp.gmail.com:587", auth, Sender, to, msg)
	if err != nil {
		return *NewString(err.Error())
	}

	return *NewString("Email sent successfully !")
}
