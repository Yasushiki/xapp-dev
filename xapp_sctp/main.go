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
	ranFunctionID = 2
)

var subResposta = make(chan struct{}, 1)

func sinalizarResposta() {
	select {
	case subResposta <- struct{}{}:
	default:
	}
}

type XappSCTP struct {
	e2ap *control.E2ap
}

func (e *XappSCTP) Consume(msg *xapp.RMRParams) (err error) {
	switch msg.Mtype {
	case 12050: // RIC_INDICATION (ou xapp.RIC_INDICATION)
		fmt.Printf("#### Métrica recebida da DU. Tamanho: %d bytes\n", msg.PayloadLen)

		decodedMsg, err := e.e2ap.GetIndicationMessage(msg.Payload)
		if err != nil {
			fmt.Printf("#### [ERRO] Erro ao decodificar RIC_INDICATION: %v\n", err)
		} else {
			fmt.Printf("#### [SUCESSO] RIC_INDICATION decodificada: %+v\n", decodedMsg)
			fmt.Printf("#### IndMessage hex: %s\n", hex.EncodeToString(decodedMsg.IndMessage))
			fmt.Printf("#### IndHeader hex:  %s\n", hex.EncodeToString(decodedMsg.IndHeader))
		}

	case 12011: // RIC_SUB_RESP
		fmt.Printf("#### [SUCESSO] Assinatura confirmada pela DU\n")
		sinalizarResposta()
	case 12012: // RIC_SUB_FAILURE
		fmt.Printf("#### [ERRO] Erro da assinatura na DU | Payload (hex): %s\n", hex.EncodeToString(msg.Payload))
		sinalizarResposta()

	default:
		fmt.Printf("#### Mensagem RMR recebida (Tipo: %d)\n", msg.Mtype)
	}
	return nil
}

func montarPayload(e2ap *control.E2ap, seq uint16) ([]byte, error) {
	// Aloca um buffer inicial para o Cgo preencher o payload codificado
	payloadBuf := make([]byte, 2048)

	var ricRequestorID uint16 = 42069
	var ricRequestSequenceNumber uint16 = seq
	var ranFuncID uint16 = ranFunctionID // 2

	// Event Trigger Definition - gerado com gerar_kpm.py
	eventTriggerDefinition := []byte{0x08, 0x03, 0xe7}
	eventTriggerSize := len(eventTriggerDefinition)

	// Parâmetros das Ações (1 ação configurada)
	actionCount := 1
	actionIds := []int64{1}
	actionTypes := []int64{0} // 0 = ActionType Report

	// Action Definition Byte - gerado com gerar_kpm.py
	actionDefBytes := []byte{0x00, 0x01, 0x04, 0x80, 0x6d, 0x00, 0x00, 0x00, 0x38, 0x40, 0x01, 0x00, 0x04, 0x01, 0x01, 0x00, 0x00, 0x04, 0x01, 0x00, 0x44, 0x52, 0x42, 0x2e, 0x52, 0x6c, 0x63, 0x53, 0x64, 0x75, 0x44, 0x65, 0x6c, 0x61, 0x79, 0x44, 0x6c, 0x01, 0x20, 0x00, 0x00, 0x00, 0xa0, 0x44, 0x52, 0x42, 0x2e, 0x55, 0x45, 0x54, 0x68, 0x70, 0x44, 0x6c, 0x01, 0x20, 0x00, 0x00, 0x00, 0xa0, 0x44, 0x52, 0x42, 0x2e, 0x55, 0x45, 0x54, 0x68, 0x70, 0x55, 0x6c, 0x01, 0x20, 0x00, 0x00, 0x00, 0xb0, 0x52, 0x52, 0x55, 0x2e, 0x50, 0x72, 0x62, 0x54, 0x6f, 0x74, 0x44, 0x6c, 0x01, 0x20, 0x00, 0x00, 0x00, 0xb0, 0x52, 0x52, 0x55, 0x2e, 0x50, 0x72, 0x62, 0x54, 0x6f, 0x74, 0x55, 0x6c, 0x01, 0x20, 0x00, 0x00, 0x40, 0x03, 0xe7}

	actionDefinitions := []control.ActionDefinition{
		{
			Buf:  actionDefBytes,
			Size: len(actionDefBytes),
		},
	}

	subsequentActions := make([]control.SubsequentAction, actionCount)

	// Chamada do wrapper Cgo
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

	fmt.Printf("#### Aguardando 15s para o registro e para as rotas\n")
	time.Sleep(15 * time.Second)

	for tentativa := 0; tentativa < 5; tentativa++ {
		seq := uint16(1101 + tentativa)

		payload, err := montarPayload(xAppInst.e2ap, seq)
		if err != nil {
			fmt.Printf("#### [ERRO] Falha ao codificar RICsubscriptionRequest: %v\n", err)
			return
		}
		fmt.Printf("#### RICsubscriptionRequest tentativa %d (%d bytes): %s\n",
			tentativa+1, len(payload), hex.EncodeToString(payload))

		fmt.Printf("#### Enviando RIC_SUB_REQ (12010)\n")

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

		select {
		case <-subResposta:
			fmt.Printf("#### [SUCESSO] Resposta recebida, encerrando tentativas\n")
			return
		case <-time.After(20 * time.Second):
			fmt.Printf("#### Sem resposta em 20s, nova tenativa\n")
		}
	}

	fmt.Printf("#### [ERRO] Sem resposta após 5 tentativas. Verificar logs do submgr e do e2term\n")

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
