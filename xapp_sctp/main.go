package main

import (
	"encoding/hex"
	"fmt"
	"time"

	// xapp-frame
	"gerrit.o-ran-sc.org/r/ric-plt/xapp-frame/pkg/xapp"

	// wrapper do ASN.1 pro Go (copiado do kpimon-go)
	"xapp_sctp/control"
)

const (
	ranName       = "gnb_001_001_00000e00"
	ranFunctionID = 0
)

type XappSCTP struct {
	e2ap *control.E2ap
}

func (e *XappSCTP) Consume(msg *xapp.RMRParams) (err error) {
	switch msg.Mtype {
	case 12050: // RIC_INDICATION (ou xapp.RIC_INDICATION)
		fmt.Printf("#### Métrica recebida do e2sim. Tamanho: %d bytes\n", msg.PayloadLen)

		decodedMsg, err := e.e2ap.GetIndicationMessage(msg.Payload)
		if err != nil {
			fmt.Printf("#### [ERRO] Erro ao decodificar RIC_INDICATION: %v\n", err)
		} else {
			fmt.Printf("#### [SUCESSO] RIC_INDICATION decodificada: %+v\n", decodedMsg)
		}

	case 12011: // RIC_SUB_RESP
		fmt.Printf("#### [SUCESSO] Assinatura confirmada pelo e2sim\n")
	case 12012: // RIC_SUB_FAILURE
		fmt.Printf("#### [ERRO] Erro da assinatura no e2sim | Payload (hex): %s\n", hex.EncodeToString(msg.Payload))
	default:
		fmt.Printf("#### Mensagem RMR recebida (Tipo: %d)\n", msg.Mtype)
	}
	return nil
}

func montarPayload(e2ap *control.E2ap) ([]byte, error) {
	// 1. Aloca um buffer inicial para o Cgo preencher o payload codificado
	payloadBuf := make([]byte, 2048)

	var ricRequestorID uint16 = 42069
	var ricRequestSequenceNumber uint16 = 1101
	var ranFuncID uint16 = ranFunctionID // 0

	// 2. Buffer do Event Trigger Definition (não pode ser nil ou vazio)
	eventTriggerDefinition := []byte{0x00, 0x00, 0x04, 0x00}
	eventTriggerSize := len(eventTriggerDefinition)

	// 3. Parâmetros das Ações (1 ação configurada)
	actionCount := 1
	actionIds := []int64{1}
	actionTypes := []int64{0} // 0 = ActionType Report

	// 4. FIX: Define a variável de bytes e a passa para o slice, garantindo Size > 0
	actionDefBytes := []byte{0x00, 0x01}

	actionDefinitions := []control.ActionDefinition{
		{
			Buf:  actionDefBytes,
			Size: len(actionDefBytes),
		},
	}

	subsequentActions := make([]control.SubsequentAction, actionCount)

	// 5. Chamada do wrapper Cgo
	payload, err := e2ap.SetSubscriptionRequestPayload(
		payloadBuf,
		ricRequestorID,
		ricRequestSequenceNumber,
		ranFuncID,
		eventTriggerDefinition,
		eventTriggerSize,
		actionCount,
		actionIds,
		actionTypes,
		actionDefinitions,
		subsequentActions,
	)
	if err != nil {
		return nil, fmt.Errorf("falha no SetSubscriptionRequestPayload: %w", err)
	}

	return payload, nil
}

func enviarAssinatura() {
	for !xapp.IsReady() {
		fmt.Printf("#### Aguardando RMR\n")
		time.Sleep(1 * time.Second)
	}

	payload, err := montarPayload(xAppInst.e2ap)
	if err != nil {
		fmt.Printf("#### [ERRO] Falha ao codificar RICsubscriptionRequest: %v\n", err)
		return
	}
	fmt.Printf("#### RICsubscriptionRequest (%d bytes): %s\n", len(payload), hex.EncodeToString(payload))

	fmt.Printf("#### Enviando RIC_SUB_REQ (12010) para o e2sim\n")

	params := &xapp.RMRParams{
		Mtype:      12010, // RIC_SUB_REQ
		SubId:      -1,
		Payload:    payload,
		PayloadLen: len(payload),
		Meid:       &xapp.RMRMeid{RanName: ranName},
	}

	if ok := xapp.Rmr.SendMsg(params); !ok {
		fmt.Printf("#### [ERRO] Falha ao enviar mensagem RMR (SendMsg retornou false)\n")
	} else {
		fmt.Printf("#### [SUCESSO] Mensagem enviada com sucesso ao RMR\n")
	}
}

var xAppInst *XappSCTP

func main() {
	fmt.Printf("#### INICIANDO xApp com suporte Cgo E2AP ####\n")
	xAppInst = &XappSCTP{
		e2ap: &control.E2ap{},
	}
	go enviarAssinatura()
	xapp.Run(xAppInst)
}
