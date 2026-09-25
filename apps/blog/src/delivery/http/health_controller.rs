use axum::{
  Router,
  routing::{get}
};

use std::sync::Arc;

#[derive(Deserialize, Serialize)]
pub struct User {
    database: bool,
}


pub struct HealthController{}

impl HealthController{
  pub fn router() -> Router{
    Router::New().route("/ping", get())
  }

  async fn ping() {

  }
}
