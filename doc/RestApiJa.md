# Francine REST API
* newSession (POST /sessions)
  * 新たにセッションを作成
  * 入力: JSON
      * InputJson (string): メイン入力ファイルのJSON名
  * 出力: JSON
      * SessionId (string): セッションのID

* editResource (PUT /sessions/:sessionId/resources/:resourceName)
  * リソースを追加・編集
  * 入力: binary
    * resourceNameがそのままファイル名として扱われる（サブディレクトリも可能）
  * 出力: JSON
    * Status (string): 成功したら"Ok"
    * Name (string): リソースのファイル名
    * Hash (string): リソースのSHA256ハッシュ

* newRenderer (POST /sessions/:sessionId/renders)
  * レンダリングを実行（レンダリングセッションを発行）
  * 入力: なし
    * レンダリングが完了するまでブロックする
      * ブロックしないオプションが追加される予定
  * 出力
    * 成功した場合: jpegファイル
    * 失敗した場合: JSON
      * Status (string): 常に"LinkError"
      * Log (string): エラーの詳細

* 追加予定のAPI
  * editSession (PUT /sessions/:sessionId)
    * newSessionに同じ
  * deleteSession (DELETE /sessions/:sessionId)
  * poll (GET /sessions/:sessionId/renders/:renderId/poll)
    * newRendererに非同期オプションが実装された時のポーリング
  * websocket
    * WebSocket経由で全てのAPIを発行できるようにする


