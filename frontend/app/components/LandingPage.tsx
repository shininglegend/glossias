import { Link } from "react-router";
import { SignUpButton, SignInButton } from "@clerk/react-router";

export function LandingPage() {
  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-50 to-blue-50">
      {/* Hero Section */}
      <div className="relative overflow-hidden">
        <div className="absolute inset-0 bg-gradient-to-r from-blue-600/10 to-purple-600/10"></div>
        <div className="relative max-w-6xl mx-auto px-4 py-24">
          <div className="text-center">
            <h1 className="text-5xl md:text-7xl font-bold tracking-tight text-slate-900 mb-8">
              Master Languages Through
              <span className="block text-transparent bg-clip-text bg-gradient-to-r from-blue-600 to-purple-600">
                Interactive Stories
              </span>
            </h1>
            <p className="text-xl md:text-2xl text-slate-600 mb-12 max-w-3xl mx-auto leading-relaxed">
              Immerse yourself in carefully crafted stories with integrated
              audio, vocabulary guides, and grammar explanations. Learn
              naturally through context and repetition.
            </p>
          </div>
        </div>
      </div>

      {/* Features Grid */}
      <div className="max-w-6xl mx-auto px-4 py-16">
        <div className="grid md:grid-cols-2 lg:grid-cols-4 gap-8">
          <div className="bg-white rounded-2xl p-8 shadow-lg hover:shadow-xl transition-shadow">
            <div className="w-12 h-12 bg-blue-100 rounded-lg flex items-center justify-center mb-6">
              <span className="material-icons text-blue-600 text-2xl">
                play_circle
              </span>
            </div>
            <h3 className="text-xl font-semibold text-slate-900 mb-4">
              Video Stories
            </h3>
            <p className="text-slate-600">
              Watch generated video stories with fluent speakers. Learn through
              visual context and immersive storytelling experiences.
            </p>
          </div>

          <div className="bg-white rounded-2xl p-8 shadow-lg hover:shadow-xl transition-shadow">
            <div className="w-12 h-12 bg-purple-100 rounded-lg flex items-center justify-center mb-6">
              <span className="material-icons text-purple-600 text-2xl">
                search
              </span>
            </div>
            <h3 className="text-xl font-semibold text-slate-900 mb-4">
              Grammar Identification
            </h3>
            <p className="text-slate-600">
              Identify and learn grammar points directly within story contexts.
              Master patterns through interactive exercises.
            </p>
          </div>

          <div className="bg-white rounded-2xl p-8 shadow-lg hover:shadow-xl transition-shadow">
            <div className="w-12 h-12 bg-green-100 rounded-lg flex items-center justify-center mb-6">
              <span className="material-icons text-green-600 text-2xl">
                hearing
              </span>
            </div>
            <h3 className="text-xl font-semibold text-slate-900 mb-4">
              Audio Vocabulary
            </h3>
            <p className="text-slate-600">
              Select correct vocabulary based on audio prompts. Build listening
              comprehension and word recognition skills.
            </p>
          </div>

          <div className="bg-white rounded-2xl p-8 shadow-lg hover:shadow-xl transition-shadow">
            <div className="w-12 h-12 bg-orange-100 rounded-lg flex items-center justify-center mb-6">
              <span className="material-icons text-orange-600 text-2xl">
                leaderboard
              </span>
            </div>
            <h3 className="text-xl font-semibold text-slate-900 mb-4">
              Leaderboard
            </h3>
            <p className="text-slate-600">
              Compete with other learners and track your progress. Stay
              motivated with achievements and friendly competition.
            </p>
          </div>
        </div>
      </div>

      {/* CTA Section */}
      <div className="bg-slate-900 text-white">
        <div className="max-w-4xl mx-auto px-4 py-20 text-center">
          <h2 className="text-3xl md:text-4xl font-bold mb-6">
            Ready to Transform Your Language Learning?
          </h2>
          <p className="text-xl text-slate-300 mb-10">
            Join a handful of learners who are mastering new languages through
            our interactive story platform.
          </p>
          <div className="flex flex-col sm:flex-row gap-4 justify-center">
            <SignUpButton>
              <button className="bg-blue-600 hover:bg-blue-700 text-white px-8 py-4 rounded-lg text-lg font-semibold transition-colors">
                Start Learning Free
              </button>
            </SignUpButton>
            <SignInButton>
              <button className="bg-transparent border-2 border-slate-300 hover:bg-white hover:text-slate-900 text-white px-8 py-4 rounded-lg text-lg font-semibold transition-colors">
                Sign In
              </button>
            </SignInButton>
          </div>
        </div>
      </div>
    </div>
  );
}
