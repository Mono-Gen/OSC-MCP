# OSC MCP サーバー 導入・機能マニュアル

本ドキュメントは、OSC MCP サーバー (`osc-mcp`) を導入して起動するための手順、および提供される全機能（MCPツール、リソース）の使い方をまとめた、**初心者向けのガイドブック**です。
プログラミングやネットワークに慣れていない方でも、AIアシスタントと協力しながらOSC機器の制御ができるよう、分かりやすく解説しています。

---

## 🔰 はじめに：基本コンセプトと用語集

本サーバーやAIを使って開発を始める前に、まずは基本となる言葉を整理しましょう！

*   **OSC（Open Sound Control）**：電子楽器や音響照明機器、コンピュータ間で音楽や制御データをやり取りするためのシンプルな通信ルールです。VRChatのアバター制御やMax/MSP、TouchOSCなどで広く使われています。
*   **UDP（User Datagram Protocol）**：インターネットなどのネットワークでデータをすばやく送受信するための仕組みです。OSCはこのUDPという通信路を使ってパケットをやり取りします。
*   **MCP（Model Context Protocol）**：AI（ClaudeやAntigravityなど）が、あなたのパソコンやOSC対応アプリと直接通信して、設定を読み取ったりメッセージを送信したりするための仕組みです。
*   **OSC Address（アドレス）**：制御したい項目を特定するための「フォルダのような目印」です。必ず `/` から開始します（例：`/avatar/parameters/Mute`）。
*   **Type Tag（型タグ）**：送信するデータが整数（`int`）、実数（`float`）、文字列（`string`）、真偽値（`bool`）のどれであるかを示すラベルです。必ず `,` から開始します。
*   **Bundle（バンドル）**：複数のOSCメッセージとタイムスタンプを1つにパックした「小包」のようなデータ構造です。

---

## 導入と起動の手順

### 1. 配布ファイルの構成
実行に必要なファイルは以下のバイナリファイルです。
*   `osc-mcp.exe`（またはお使いのOS用のバイナリ）

### 2. 各OSでのコンパイルと実行方法

#### ■ Windows の場合
1. ソースコードから実行ファイルをビルドする場合：
   ```powershell
   $env:GOOS="windows"
   $env:GOARCH="amd64"
   go build -o osc-mcp.exe main.go
   ```
2. コマンドプロンプトやPowerShellから実行します。
   ```powershell
   .\osc-mcp.exe
   ```

#### ■ Mac の場合 (Apple Silicon/Intel)
1. ソースコードから実行ファイルをビルドする場合：
   - Apple Silicon (M1/M2/M3等):
     ```bash
     GOOS=darwin GOARCH=arm64 go build -o osc-mcp main.go
     ```
   - Intel製CPU:
     ```bash
     GOOS=darwin GOARCH=amd64 go build -o osc-mcp main.go
     ```
2. **実行権限の付与 (初回のみ)**:
   ターミナルを開き、配置したフォルダに移動して以下のコマンドを実行してプログラムに実行権限を与えます。
   ```bash
   chmod +x ./osc-mcp
   ```
3. **開発元未確認警告の回避 (初回のみ)**:
   Finderでバイナリファイルを **右クリック（または Ctrl を押しながらクリック）して「開く」** を選択します。警告画面が出ますが、「開く」を選択することでMacに実行を許可させることができます。
4. ターミナルから起動します。

---

### 3. 各種AIアシスタント (Claude, Antigravity等) との連携

本サーバーをAIアシスタントに登録することで、AIがあなたの指示に従って自動的にOSCパラメータの調整を行ったり、機器への送受信を行ったりできるようになります。

#### 1. 設定ファイルの場所
お使いのツールとOSに合わせて、以下の設定ファイルをテキストエディタで編集します。

*   **Claude Desktop**
    *   **Windows**: `%APPDATA%\Claude\claude_desktop_config.json`
    *   **Mac**: `~/Library/Application Support/Claude/claude_desktop_config.json`
*   **Antigravity / MCP対応AIエージェント**
    *   **共通**: `~/.gemini/antigravity-ide/mcp_settings.json` など

#### 2. 設定ファイルの記述例

##### ■ Windows の場合
Windows環境では、以下のように `cmd.exe /c` を経由して起動させることで、安全かつ確実に動作させることができます。

```json
{
  "mcpServers": {
    "osc-mcp": {
      "command": "cmd.exe",
      "args": [
        "/c",
        "C:\\path\\to\\osc-mcp.exe"
      ],
      "cwd": "C:\\path\\to\\project_directory"
    }
  }
}
```
※ パス区切り文字は、必ず **バックスラッシュ（円マーク）を2つ重ねた `\\`** で記述してください。

##### ■ Mac の場合
Macでは、バイナリファイルを直接指定して動作させることができます。

```json
{
  "mcpServers": {
    "osc-mcp": {
      "command": "/path/to/osc-mcp",
      "args": [],
      "cwd": "/path/to/project_directory"
    }
  }
}
```

---

## 提供される機能（MCPツール・リソース）一覧

AIアシスタントは、以下のツールやリソースを自動的に使い分けます。

### 1. コントロール制御ツール (Tools)

*   **`send_osc`**
    *   **何をする？**: 指定された宛先（ホスト、ポート、OSCアドレス）に対してOSCメッセージを送信します。
    *   **引数**:
      - `host` (string, 任意): 宛先IPアドレス (デフォルト: `127.0.0.1`)。
      - `port` (integer, 必須): 宛先UDPポート (例: VRChat の場合は `9000`)。
      - `address` (string, 必須): OSCアドレス (必ず `/` から開始。例: `/avatar/parameters/Mute`)。
      - `args` (array, 任意): 送信する引数のリスト。各項目は以下を含みます：
        - `type` (string): `"int"`, `"float"`, `"string"`, `"bool"` のいずれか。
        - `value` (any): 送信する値。

*   **`start_listen_osc`**
    *   **何をする？**: 指定されたポートでUDPソケットを開き、OSCメッセージの待ち受けを開始します。
    *   **引数**:
      - `port` (integer, 必須): 待ち受けを行うローカルUDPポート番号（例：VRChatからの受信は `9001`）。
      - `host` (string, 任意): 待ち受けを行うローカルIPアドレス（デフォルト: `127.0.0.1`）。外部デバイスからメッセージを受ける場合は `0.0.0.0` を明示します。
    *   **仕様**: バックグラウンドで受信ループを起動します。既に同一ポートで受信中の場合は、ソケットを再作成せず動作を維持します。ポートごとに最大100件の受信メッセージをリングバッファにキャッシュします。

*   **`stop_listen_osc`**
    *   **何をする？**: 指定されたポートの待ち受けを終了し、UDPソケットをクローズして解放します。
    *   **引数**:
      - `port` (integer, 必須): 停止するポート番号。

### 2. 受信履歴リソース (Resources)

*   **`osc://messages/{port}/recent`**
    *   **何をする？**: 指定されたポートで最近受信したOSCメッセージ履歴（最大100件）を取得します。
    *   **仕様**: 受信キャッシュから履歴をJSON形式で返却します。タイムスタンプやパケット送信元の情報も含まれます。

---

## 困ったときは？（トラブルシューティング）

*   **Q. AIが「ポートをバインドできない」とエラーを出します**
    *   **A.** `start_listen_osc` を呼び出した際にエラーが発生する場合、指定したポートが他のプログラムによってすでに使用されている可能性があります。以下のコマンド等でポートを使用しているプロセスを特定し、終了してください。
        - **Windows (Command Prompt)**:
          ```cmd
          netstat -ano | findstr <ポート番号>
          ```
*   **Q. AIアシスタントが突然クラッシュします**
    *   **A.** サーバーのデバッグ用に `fmt.Println` などを直接ソースコードに追加して標準出力に不正な文字列が混ざると、JSON-RPCプロトコルが壊れ、クライアントがクラッシュします。デバッグ出力には必ず `os.Stderr` (標準エラー出力) を使用してください。
