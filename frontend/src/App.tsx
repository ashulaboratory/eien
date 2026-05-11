import { useEffect, useState } from 'react'

// バックエンドからのレスポンスの型を定義
interface PingResponse {
  message: string
  db_time: string
}

function App() {
  // useState でstate管理（初期値はnull）
  const [ping, setPing] = useState<PingResponse | null>(null)
  const [error, setError] = useState<string | null>(null)

  // useEffect で「コンポーネントマウント時に1回」fetchを実行
  useEffect(() => {
    fetch('/api/ping')
      .then((res) => {
        if (!res.ok) throw new Error(`HTTP ${res.status}`)
        return res.json()
      })
      .then((data: PingResponse) => setPing(data))
      .catch((err: Error) => setError(err.message))
  }, []) // 空配列 = 「1回だけ実行」

  return (
    <div className="min-h-screen bg-gray-100 flex items-center justify-center">
      <div className="bg-white rounded-lg shadow-lg p-8 max-w-md w-full">
        <h1 className="text-4xl font-bold text-blue-600">Eien</h1>
        <p className="mt-4 text-gray-700">永遠の仲間とのSNS</p>

        <div className="mt-6 border-t pt-4">
          <h2 className="text-sm font-semibold text-gray-700 mb-2">
            バックエンド疎通テスト
          </h2>
          {error && (
            <p className="text-red-600 text-sm">❌ エラー: {error}</p>
          )}
          {ping && (
            <div className="text-sm space-y-1">
              <p className="text-green-600">✓ 接続成功</p>
              <p className="text-gray-600">
                message: <span className="font-mono">{ping.message}</span>
              </p>
              <p className="text-gray-600">
                db_time:{' '}
                <span className="font-mono text-xs">{ping.db_time}</span>
              </p>
            </div>
          )}
          {!ping && !error && (
            <p className="text-gray-400 text-sm">読み込み中...</p>
          )}
        </div>
      </div>
    </div>
  )
}

export default App