package main

import (
	"context"
	"fmt"
	"io"
)

import (
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
)

import (
	"github.com/google/uuid"
	"github.com/hasuburero/util/log"
)

var (
	Images = []string{
		"192.168.128.2:5000/myfaas-test-python:3.15.0a2",
		"192.168.128.2:5000/myfaas-test-python:3.14.0",
		"192.168.128.2:5000/myfaas-test-python:3.13.9",
		"192.168.128.2:5000/myfaas-test-python:3.12.12",
		"192.168.128.2:5000/myfaas-test-python:3.11.14",
		"192.168.128.2:5000/myfaas-test-python:3.10.19",
	}
)

func main() {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return
	}
	defer cli.Close()
	containers, err := cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		log.PrintLog(err.Error(), "ContainerList")
		return
	}

	for _, cont := range containers {
		// 強制削除オプション
		err := cli.ContainerRemove(ctx, cont.ID, container.RemoveOptions{
			Force: true, // 実行中のコンテナも強制的に停止して削除
		})
		if err != nil {
			fmt.Printf("コンテナ %s の削除に失敗: %v\n", cont.ID[:12], err)
		} else {
			fmt.Printf("  コンテナ %s を削除しました。\n", cont.ID[:12])
		}
	}

	// 2. すべてのイメージを削除
	fmt.Println("2. すべてのイメージを削除中...")
	images, err := cli.ImageList(ctx, image.ListOptions{All: true})
	if err != nil {
		log.PrintLog(err.Error(), "Imagelist")
		return
	}

	for _, img := range images {
		// 使用中のイメージも強制的に削除（削除対象のイメージに依存するコンテナは既に削除済み）
		_, err := cli.ImageRemove(ctx, img.ID, image.RemoveOptions{
			Force: true,
		})
		if err != nil {
			// イメージが他のコンテナ/イメージで参照されている場合など、エラーになる可能性あり
			fmt.Printf("イメージ %s の削除に失敗: %v\n", img.ID[7:19], err)
		} else {
			fmt.Printf("  イメージ %s を削除しました。\n", img.ID[7:19])
		}
	}
	for {
		for _, image_name := range Images {

			pullResp, err := cli.ImagePull(ctx, image_name, image.PullOptions{})
			if err != nil {
				return
			}
			defer pullResp.Close()

			_, _ = io.ReadAll(pullResp)

			id_buf, err := uuid.NewRandom()
			if err != nil {
				return
			}

			resp, err := cli.ContainerCreate(
				ctx,
				&container.Config{
					Image: image_name,
					Cmd:   []string{"bash"},
					Tty:   true,
					Env:   []string{},
				},
				&container.HostConfig{
					AutoRemove: true,
				},
				nil,
				nil,
				id_buf.String(),
			)

			if err != nil {
				fmt.Println(err)
				return
			}

			err = cli.ContainerStart(ctx, resp.ID, container.StartOptions{})
			if err != nil {
				log.PrintLog(err.Error(), "Container Start")
				return
			}

			err = cli.ContainerRemove(ctx, resp.ID, container.RemoveOptions{
				RemoveVolumes: true,
				Force:         true,
				RemoveLinks:   false,
			})

			if err != nil {
				log.PrintLog(err.Error(), "ContainerRemove")
				return
			}

			_, _ = cli.ImageRemove(ctx, image_name, image.RemoveOptions{
				Force:         false,
				PruneChildren: true,
			})
		}
	}
}
