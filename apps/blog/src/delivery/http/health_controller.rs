use axum::{
  Json, Router, routing::get
};
use serde::{Deserialize, Serialize};

#[derive(Serialize, Deserialize, Debug)]
pub struct Status {
  database: bool,
  database_error: String,
}

#[derive(Debug)]
pub struct HealthController{}

impl HealthController{

  pub fn routes() -> Router {
    Router::new().route("/ping", get(Self::ping))
  }

  async fn ping() -> Json<Status> {
    Json(Status { database: false, database_error: String::new() })
  }
}
