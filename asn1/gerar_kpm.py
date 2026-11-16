#!/usr/bin/env python3
"""
Gera os bytes do Event Trigger Definition e do Action Definition (Format 4)
do E2SM-KPM v2.03, em PER alinhado, para colar no main.go do xApp.

Uso (na sua maquina, uma vez):
    pip install asn1tools
    python3 gerar_kpm.py --asn openair2/E2AP/flexric/src/sm/kpm_sm/kpm_sm_v02.03/ie/e2sm_kpm_v02.03_standard.asn1

Para mudar o que voce pede, altere os argumentos, por exemplo:
    python3 gerar_kpm.py --asn ARQUIVO.asn1 --metricas DRB.UEThpDl RRU.PrbTotDl --periodo-ms 2000

ATENCAO: este script nao foi testado por quem o escreveu (o ambiente dele nao
tinha o asn1tools). Se der erro, copie a mensagem e mande de volta.
"""
import argparse
import sys

import asn1tools

# Metricas que a DU do OAI sabe calcular (lista kpm_meas_du em ran_func_kpm.c).
METRICAS_DU = [
    "DRB.RlcSduDelayDl",
    "DRB.UEThpDl",
    "DRB.UEThpUl",
    "RRU.PrbTotDl",
    "RRU.PrbTotUl",
]


def como_go(nome, dados):
    """Formata os bytes como um slice Go."""
    corpo = ", ".join(f"0x{b:02x}" for b in dados)
    return f"{nome} := []byte{{{corpo}}} // {len(dados)} bytes"


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawTextHelpFormatter)
    ap.add_argument("--asn", required=True, help="caminho do e2sm_kpm_v02.03_standard.asn1")
    ap.add_argument("--metricas", nargs="+", default=METRICAS_DU,
                    help="nomes das metricas (padrao: as 5 que a DU suporta)")
    ap.add_argument("--periodo-ms", type=int, default=1000,
                    help="de quanto em quanto tempo a DU envia o relatorio (Event Trigger)")
    ap.add_argument("--granularidade-ms", type=int, default=1000,
                    help="granularidade da medicao (Action Definition)")
    ap.add_argument("--sst", type=int, default=1,
                    help="SST do slice usado para filtrar os UEs (1 byte, sem SD)")
    args = ap.parse_args()

    # Avisa se alguma metrica nao existe na DU, porque ela nao saberia calcular.
    desconhecidas = [m for m in args.metricas if m not in METRICAS_DU]
    if desconhecidas:
        print(f"AVISO: a DU nao lista estas metricas: {desconhecidas}", file=sys.stderr)

    try:
        # "per" no asn1tools = PER alinhado (o que a DU exige).
        kpm = asn1tools.compile_files(args.asn, "per")
    except Exception as e:
        print(f"Erro ao ler o .asn1: {e}", file=sys.stderr)
        print("Dica: se reclamar de comentario, apague os trechos '-- This IE shall not be "
              "present ... --' dentro de BinRangeDefinition, numa COPIA do arquivo.",
              file=sys.stderr)
        sys.exit(1)

    # --- Event Trigger Definition: "me mande relatorios a cada X ms" ---
    event_trigger = {
        "eventDefinition-formats": (
            "eventDefinition-Format1",
            {"reportingPeriod": args.periodo_ms},
        )
    }

    # --- Action Definition, Style 4 / Format 4 ---
    # 1) Filtro de UEs: so os UEs do slice com este SST (unica condicao que o OAI implementa).
    # 2) Lista de metricas (Format 1 embutido).
    action_def = {
        "ric-Style-Type": 4,
        "actionDefinition-formats": (
            "actionDefinition-Format4",
            {
                "matchingUeCondList": [
                    {
                        "testCondInfo": {
                            "testType": ("sNSSAI", "true"),
                            "testExpr": "equal",
                            "testValue": ("valueOctS", bytes([args.sst])),
                        }
                    }
                ],
                "subscriptionInfo": {
                    "measInfoList": [
                        {
                            "measType": ("measName", nome),
                            # o ASN.1 exige ao menos 1 label; "noLabel" = sem rotulo
                            "labelInfoList": [{"measLabel": {"noLabel": "true"}}],
                        }
                        for nome in args.metricas
                    ],
                    "granulPeriod": args.granularidade_ms,
                },
            },
        ),
    }

    et_bytes = kpm.encode("E2SM-KPM-EventTriggerDefinition", event_trigger)
    ad_bytes = kpm.encode("E2SM-KPM-ActionDefinition", action_def)

    # Conferencia: decodifica de volta para garantir que os bytes fazem sentido.
    kpm.decode("E2SM-KPM-EventTriggerDefinition", et_bytes)
    kpm.decode("E2SM-KPM-ActionDefinition", ad_bytes)

    print("// Cole no main.go, dentro de montarPayload:")
    print(como_go("eventTriggerDefinition", et_bytes))
    print(como_go("actionDefBytes", ad_bytes))
    print()
    print("// hex (para conferir no Wireshark / logs):")
    print("event trigger :", et_bytes.hex())
    print("action def    :", ad_bytes.hex())


if __name__ == "__main__":
    main()
