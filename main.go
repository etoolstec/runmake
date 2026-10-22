package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
)

type Job struct {
	Name   string // nome para identificar nos logs
	Dir    string // pasta onde está o Makefile
	File   string // nome do arquivo (ex: Makefile, build.mk)
	Target string // alvo do make (ex: build, test). Vazio = alvo padrão
}

type Result struct {
	Job Job
	Err error
}

func runMake(ctx context.Context, job Job) error {
	args := []string{"-f", job.File}
	if job.Target != "" {
		args = append(args, job.Target)
	}

	cmd := exec.CommandContext(ctx, "make", args...)
	cmd.Dir = job.Dir // executa dentro da pasta do script
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("[%s] iniciando (dir=%s, file=%s)\n", job.Name, job.Dir, job.File)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("[%s] falhou: %w", job.Name, err)
	}

	fmt.Printf("[%s] finalizado com sucesso\n", job.Name)
	return nil
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	data, err := os.ReadFile("./jobs.json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "erro ao ler o arquivo de jobs: %v\n", err)
		os.Exit(1)
	}

	var jobs []Job
	if err := json.Unmarshal(data, &jobs); err != nil {
		fmt.Fprintf(os.Stderr, "erro ao analisar o arquivo de jobs: %v\n", err)
		os.Exit(1)
	}

	var wg sync.WaitGroup
	results := make(chan Result, len(jobs))

	for _, job := range jobs {
		wg.Add(1)
		go func(j Job) {
			defer wg.Done()
			results <- Result{Job: j, Err: runMake(ctx, j)}
		}(job)
	}

	wg.Wait()
	close(results)

	failed := false
	for r := range results {
		if r.Err != nil {
			fmt.Fprintln(os.Stderr, r.Err)
			failed = true
		}
	}

	if failed {
		os.Exit(1)
	}
	fmt.Println("todos os scripts terminaram com sucesso")
}
