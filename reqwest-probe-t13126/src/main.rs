use std::error::Error;

#[tokio::main(flavor = "current_thread")]
async fn main() {
    let client = reqwest::Client::builder().user_agent("reqwest-probe").build().unwrap();
    for url in std::env::args().skip(1) {
        match client.get(&url).send().await {
            Ok(resp) => println!("REPRO ok url={url} status={}", resp.status()),
            Err(e) => {
                println!("REPRO err url={url} display={e}");
                println!("REPRO err debug={e:?}");
                let mut source = e.source();
                while let Some(s) = source {
                    println!("REPRO   caused by: {s}");
                    source = s.source();
                }
            }
        }
    }
}
