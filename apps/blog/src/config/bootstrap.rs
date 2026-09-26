use axum::Router;

use crate::delivery::http::health_controller::HealthController;

#[derive(Clone)]
pub struct AppState {
  pub router: Router
}

pub fn bootstrap() -> AppState{
  let router = Router::new().merge(HealthController::routes());

  return AppState { router }
}
