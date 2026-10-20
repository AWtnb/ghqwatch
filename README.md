# ghqwatch

<img src="./icon.ico" alt="up-to-date" style="width:10%"></img>
<img src="./icon-unchecked.ico" alt="unchecked" style="width:10%"></img>
<img src="./icon-out-of-date.ico" alt="behind" style="width:10%"></img>

[ghq](https://github.com/x-motemen/ghq) 管理下のリポジトリを定期チェックし、リモートより遅れている（behind）ものがあればトースト通知する Windows 用タスクトレイ常駐ツール。

## 概要

- `ghq list -p` で全リポジトリを自動検出
- `git fetch` + `git rev-list --count HEAD..@{u}` で behind 判定（4並列）
- behind があればトースト通知。同じ内容では再通知しない
- トレイメニューから手動チェックと更新（`git pull --ff-only`）
- 自動更新はしない。更新はメニューをクリックしたときだけ
- オフライン起動時は「最新」と誤表示せずエラー表示し、3分おきに再試行

## 要件

- Windows
- Go 1.25 以上（`sync.WaitGroup.Go` を使用）
- ghq, git（PATH が通っていること）
- アイコン3種（`main.go` と同じフォルダ）
	- `icon.ico` = 最新状態
	- `icon-out-of-date.ico` =  behind あり
	- `icon-unchecked.ico` = 未チェック / チェック中 / ERROR

> [!NOTE]
> アイコンは `go:embed` でビルド時に埋め込む。無いとビルドが失敗する。
> PNG から作る場合:
> 
> ```
> magick icon.png -define icon:auto-resize=256,128,64,48,32,24,16 icon.ico
> ```

## ビルドと登録

```
powershell -nop .\build.ps1
```

`go install` で GOBIN(未設定なら `%GOPATH%\bin`、通常 `%USERPROFILE%\go\bin`)に exe を配置し、スタートアップにショートカットを作って起動する。次回ログオンから自動起動。

| オプション | 内容 |
| --- | --- |
| `-Name` | exe 名・モジュール名（既定: `ghqwatch`） |
| `-NoRegister` | ビルドのみ。スタートアップに登録しない |
| `-Console` | コンソール付きでビルド。登録・起動しない。エラー出力を見たいとき |
| `-Unregister` | 登録解除（スタートアップの exe を削除。起動中なら停止）。ビルドしない |

`build.ps1` がやること:

1. アイコン3種の存在確認
2. `go.mod` が無ければ `go mod init`、続けて `go mod tidy`
3. 起動中の同名プロセスを停止
4. `go install -ldflags "-H=windowsgui"`
5. `shell:startup` に `.lnk` を作成して起動

ショートカットは GOBIN の exe を参照するので、更新時は `.\build.ps1` を再実行するだけ。
有効/無効の切り替えはタスクマネージャーの「スタートアップ アプリ」。

> [!WARNING]
> - `go install` の出力名はモジュール名で決まる。`-o` は使えない
> - `-Console` で入れた exe はショートカットの参照先と同じ。次回ログオンで黒窓付きで起動するので、通常の `.\build.ps1` で戻す

手動ビルド:

```
go build -ldflags "-H=windowsgui" -o ghqwatch.exe .
```

`-H=windowsgui` を付けないと、起動時に黒いコンソール窓が出る。

## メニュー

| 項目 | 内容 |
| --- | --- |
| ステータス行 | 最終チェック時刻と結果（`Up-to-date` / `n updates available` / `ERROR`） |
| Check now | 今すぐチェック |
| Update n | behind のリポジトリを `git pull --ff-only`（behind があるときのみ表示） |
| Quit | 終了 |

## 動作

- 起動直後に1回チェック。以降は1時間おき（起動時刻基準。スリープ中は止まる）
- 全リポジトリで `git fetch` 失敗 → ERROR 表示。3分おきに成功するまで再試行
- 一部だけ失敗 → 結果と失敗件数を表示。同様に再試行
  - 恒常的に失敗するリポジトリがあると再試行が止まらない。気になるなら `check` の `return 0 < res.failed` を `return false` にする
- upstream 未設定、detached HEAD は対象外（失敗扱いにしない）
- チェック中に次のチェックが重なったらスキップ
- `--ff-only` なので、ローカルに独自コミットがあるリポジトリは更新されない

## 設定

設定ファイルは無し。`main.go` の定数を変えて再ビルド。

```go
const (
	interval   = 12 * time.Hour   // 定期チェック間隔
	retryAfter = 3 * time.Minute // 失敗時の再試行間隔
	workers    = 4               // 並列数
)
```

間隔を短くしすぎると `git fetch` のアクセスが増える。30分〜1時間が目安。

## トラブルシューティング

| 症状 | 原因・対処 |
| --- | --- |
| 起動時に黒い窓が出る | `-H=windowsgui` なしでビルドしている。`build.ps1` を使う（`-Console` は黒窓が出る） |
| `pattern icon.ico: no matching files found` | `icon.ico` が `main.go` と同じフォルダに無い |
| exe を上書きできない | 起動中。トレイの Quit か、`build.ps1` に停止させる |
| 通知が出ない | Windows の集中モード、通知設定を確認 |
| エラー詳細を見たい | `.\build.ps1 -Console` でビルドし、ターミナルから起動 |
| Defender が `Trojan:Script/Conteban.A!ml` と誤検知する | 未署名 Go exe の ML 誤検知。フォルダを除外、または Microsoft に誤検知報告 |
| 通知が見切れる | `go run ./notifytest` でダミー通知を出せるので調整する |