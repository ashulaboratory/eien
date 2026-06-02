// レッスン1の練習：コンポーネントとJSX
// 「コンポーネント = 見た目(HTML)を返す、ただの関数」を体で覚えるためのファイル。

// (1) 大文字始まりの関数名 = これは自作コンポーネントだ、という React の約束
function Greeting() {
  // (2) 関数の中で、普通の JavaScript を実行できる。
  //     return より前なので、ここで計算や変数の準備をする。
  const now = new Date().toLocaleTimeString();

  // (3) return の中に「見た目」を書く。これが JSX。
  return (
    // (4) JSX は「1つの親タグ」で包む必要がある。だから <div> でまとめている。
    <div>
      <h2>こんにちは、村上さん</h2>

      {/* (5) { } の中は JavaScript。さっき作った now 変数を画面に埋め込む。 */}
      <p>いまの時刻は {now} です</p>
    </div>
  );
}

// (6) 他のファイルから <Greeting /> として使えるように公開(export)する
export default Greeting;
