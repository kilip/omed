use crate::config::{bootstrap::bootstrap};

pub mod config;
pub mod delivery;



#[tokio::main]
async fn main() -> Result<(), std::io::Error> {
  let state = bootstrap();

  let listener = tokio::net::TcpListener::bind("0.0.0.0:9000").await?;
  axum::serve(listener, state.router).await?;

  Ok(())
}
