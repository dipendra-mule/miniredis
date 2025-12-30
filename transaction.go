package main

type Transaction struct {
	cmds []*TxCommand
}

func NewTransaction() *Transaction {
	return &Transaction{}
}

type TxCommand struct {
	r       *Resp
	handler Handler
}
