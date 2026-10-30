package main

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// carregarEnv lê o .env da raiz do repositório (a primeira pasta acima do
// diretório atual que contém .git) e define as variáveis que ainda não
// existem no ambiente. Fora de um repositório ou sem .env, não faz nada.
func carregarEnv() error {
	raiz, ok := raizDoRepositorio()
	if !ok {
		return nil
	}
	caminho := filepath.Join(raiz, ".env")
	f, err := os.Open(caminho)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for n := 1; scanner.Scan(); n++ {
		linha := strings.TrimSpace(scanner.Text())
		if linha == "" || strings.HasPrefix(linha, "#") {
			continue
		}
		chave, valor, ok := strings.Cut(linha, "=")
		chave = strings.TrimSpace(chave)
		if !ok || chave == "" {
			return fmt.Errorf("%s:%d: linha inválida, use CHAVE=valor", caminho, n)
		}
		if _, existe := os.LookupEnv(chave); !existe {
			os.Setenv(chave, strings.TrimSpace(valor))
		}
	}
	return scanner.Err()
}

func raizDoRepositorio() (string, bool) {
	dir, err := os.Getwd()
	if err != nil {
		return "", false
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir, true
		}
		pai := filepath.Dir(dir)
		if pai == dir {
			return "", false
		}
		dir = pai
	}
}
