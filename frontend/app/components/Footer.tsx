export function Footer() {
  return (
    <div className="bg-slate-50 border-t">
      <div className="max-w-6xl mx-auto px-4 py-6">
        <div className="grid md:grid-cols-4 gap-8">
          <div className="md:col-span-2">
            <h3 className="text-2xl font-bold text-slate-900 mb-4">Glossias</h3>
            <p className="text-slate-600 mb-6">
              Interactive language learning through immersive stories targeting
              vocabulary and grammar in context.
            </p>
            <div className="flex space-x-4">
              <a
                href="https://github.com/shininglegend/glossias"
                className="text-slate-400 hover:text-slate-600 transition-colors"
              >
                <span className="sr-only">GitHub</span>
                <svg
                  className="w-6 h-6"
                  fill="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path d="M12 2C6.477 2 2 6.484 2 12.017c0 4.425 2.865 8.18 6.839 9.504.5.092.682-.217.682-.483 0-.237-.008-.868-.013-1.703-2.782.605-3.369-1.343-3.369-1.343-.454-1.158-1.11-1.466-1.11-1.466-.908-.62.069-.608.069-.608 1.003.07 1.531 1.032 1.531 1.032.892 1.53 2.341 1.088 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.113-4.555-4.951 0-1.093.39-1.988 1.029-2.688-.103-.253-.446-1.272.098-2.65 0 0 .84-.27 2.75 1.026A9.564 9.564 0 0112 6.844c.85.004 1.705.115 2.504.337 1.909-1.296 2.747-1.027 2.747-1.027.546 1.379.202 2.398.1 2.651.64.7 1.028 1.595 1.028 2.688 0 3.848-2.339 4.695-4.566 4.943.359.309.678.92.678 1.855 0 1.338-.012 2.419-.012 2.747 0 .268.18.58.688.482A10.019 10.019 0 0022 12.017C22 6.484 17.522 2 12 2z"></path>
                </svg>
              </a>
            </div>
          </div>
          <div className="grid grid-cols-2 md:col-span-2 gap-8">
            <div>
              <h4 className="text-sm font-semibold text-slate-900 uppercase tracking-wide mb-4">
                Features
              </h4>
              <ul className="space-y-2">
                <li>
                  <a
                    href="#"
                    className="text-slate-600 hover:text-slate-900 transition-colors"
                  >
                    Audio Stories
                  </a>
                </li>
                <li>
                  <a
                    href="#"
                    className="text-slate-600 hover:text-slate-900 transition-colors"
                  >
                    Vocabulary Tools
                  </a>
                </li>
                <li>
                  <a
                    href="#"
                    className="text-slate-600 hover:text-slate-900 transition-colors"
                  >
                    Grammar Guide
                  </a>
                </li>
                <li>
                  <a
                    href="#"
                    className="text-slate-600 hover:text-slate-900 transition-colors"
                  >
                    Scoreboard
                  </a>
                </li>
              </ul>
            </div>
            <div>
              <h4 className="text-sm font-semibold text-slate-900 uppercase tracking-wide mb-4">
                Support
              </h4>
              <ul className="space-y-2">
                <li>
                  <a
                    href="https://github.com/shininglegend/glossias/wiki"
                    className="text-slate-600 hover:text-slate-900 transition-colors"
                  >
                    Help Center
                  </a>
                </li>
                <li>
                  <a
                    href="https://github.com/shininglegend/glossias/issues"
                    className="text-slate-600 hover:text-slate-900 transition-colors"
                  >
                    Report a Problem
                  </a>
                </li>
                <li>
                  <a
                    href="/privacy-policy"
                    className="text-slate-600 hover:text-slate-900 transition-colors"
                  >
                    Privacy Policy
                  </a>
                </li>
                <li>
                  <a
                    href="/terms-of-service"
                    className="text-slate-600 hover:text-slate-900 transition-colors"
                  >
                    Terms of Service
                  </a>
                </li>
              </ul>
            </div>
          </div>
        </div>
        <div className="border-t border-slate-200 mt-4 pt-4 text-center">
          <p className="text-slate-600">
            &copy; 2025 Titus M. All rights reserved.
          </p>
        </div>
      </div>
    </div>
  );
}
