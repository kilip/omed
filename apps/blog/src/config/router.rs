use axum::Router;

pub fn new_router() -> Router{
  let app = Router::new();
  return app
}
