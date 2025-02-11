# main.py
from flask import Flask
from stories.routes import configure_routes
import logging, os

def create_app():
    app = Flask(__name__,
                template_folder='./src/templates',
                static_folder='./static')

    # Enhanced logging configuration
    logging.basicConfig(
        level=logging.DEBUG,
        format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
    )

    # Add error handlers
    @app.errorhandler(500)
    def handle_500(error):
        app.logger.error(f'Server error: {error}', exc_info=True)
        return f"Server error: {str(error)}", 500

    @app.errorhandler(404)
    def handle_404(error):
        app.logger.warning(f'Not found: {error}')
        return f"Not found: {str(error)}", 404

    # Register routes
    configure_routes(app)

    return app

if __name__ == '__main__':
    port = os.getenv('PORT')
    if not port:
        print("PORT environment variable not set. Exiting.")
        exit()
    app = create_app()
    app.run(port=8081, debug=True)
