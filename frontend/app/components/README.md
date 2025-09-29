# Component Style Scheme

## Layout Structure

### Header Section
- `<header>` contains title, step indicator, instructions, and action buttons
- `<h1>` for story title
- `<h2>` for step indicator (e.g., "Vocabulary Practice")
- Instructions in gray info box with icon
- Primary action buttons in header when applicable

### Content Section
- `<div className="max-w-4xl mx-auto px-5">` for main content container
- `<div className="story-lines text-2xl max-w-3xl mx-auto">` for story text

## Text Direction (RTL/LTR)
- RTL languages: `["he", "ar", "fa", "ur"]`
- Apply `dir={isRTL ? "rtl" : "ltr"}` and `className={isRTL ? "text-right" : "text-left"}`
- Check `pageData.language` or `pageData.languageCode` for language detection

## Info Boxes
```jsx
<div className="bg-gray-50 border border-gray-300 p-4 mb-4 rounded-lg text-center">
  <div className="flex items-start justify-center">
    <span className="material-icons text-gray-600 mr-2 mt-1">info</span>
    <div>
      <p className="text-gray-700 mb-2">Primary instruction</p>
      <p className="text-gray-700">Secondary instruction</p>
    </div>
  </div>
</div>
```

## Result/Status Boxes
- Success: `bg-green-50 border-l-4 border-green-400` 
- Warning: `bg-yellow-50 border-l-4 border-yellow-400`
- Info: `bg-blue-50 border-l-4 border-blue-400`

## Next/Continue Buttons
- Standard style: `inline-flex items-center px-8 py-4 bg-green-500 text-white rounded-lg hover:bg-green-600 text-lg font-semibold transition-all duration-200 shadow-lg`
- Blue variant: Replace `green-500/600` with `blue-500/600`
- Icon: `<span className="material-icons ml-2">arrow_forward</span>`
- Container: `<div className="text-center">`

## Story Text
- Container: `text-3xl` for story lines
- Individual lines: `story-line inline` class
- Line content: `line-content text-3xl` class
- Interactive elements respect RTL/LTR directionality

## Loading/Error States
- Container: `<div className="container">`
- Consistent error messages with back navigation
- Loading messages: simple text in container

## Icons
- Use Material Icons: `<span className="material-icons">icon_name</span>`
- Common icons: `info`, `translate`, `assessment`, `arrow_forward`, `play_arrow`, `pause`
