/*
 * Copyright 2018 IBM Corp. All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Javascript code for MAX Image Caption Generator Web App

// Adds a new image to the image picker from the given return json of an upload
function add_thumbnails(data) {
    var key_list = get_keys();
    for (var i = 0; i < data.length; i++) {
        file_data = data[i];
        if (key_list.includes(file_data["file_name"]) == false) {
            $('#thumbnails select').prepend($("<option></option>")
                .attr("data-img-src", file_data["file_name"])
                .attr("data-img-label", file_data["caption"])
                .attr("data-img-alt", file_data["caption"])
                .attr("value", file_data["file_name"])
                .text(file_data["caption"]));
        }
    }
}

// Returns the list of the file paths for the images (the file path is an image's key in the File Picker)
function get_keys() {
    return $('#thumbnails select').children('option').map(function () {
        return $(this).attr('data-img-src');
    }).get();
}

// Returns a list of all the words in the captions of the currently selected images
function get_words() {
    var all_words = [];
    $('#thumbnails .image_picker_selector .selected').children().filter("img").each(function () {
        capt_text = $(this).attr('alt');
        word_arr = capt_text.split(' ');
        $.merge(all_words, word_arr);
    });
    return all_words;
}

// Create word entries with weights for use in the word cloud
function get_word_entries() {
    var words = get_words();

    // common words from https://bl.ocks.org/blockspring/847a40e23f68d6d7e8b5
    var common = "poop,i,me,my,myself,we,us,our,ours,ourselves,you,your,yours,yourself,yourselves,he,him,his,himself,she,her,hers,herself,it,its,itself,they,them,their,theirs,themselves,what,which,who,whom,whose,this,that,these,those,am,is,are,was,were,be,been,being,have,has,had,having,do,does,did,doing,will,would,should,can,could,ought,i'm,you're,he's,she's,it's,we're,they're,i've,you've,we've,they've,i'd,you'd,he'd,she'd,we'd,they'd,i'll,you'll,he'll,she'll,we'll,they'll,isn't,aren't,wasn't,weren't,hasn't,haven't,hadn't,doesn't,don't,didn't,won't,wouldn't,shan't,shouldn't,can't,cannot,couldn't,mustn't,let's,that's,who's,what's,here's,there's,when's,where's,why's,how's,a,an,the,and,but,if,or,because,as,until,while,of,at,by,for,with,about,against,between,into,through,during,before,after,above,below,to,from,up,upon,down,in,out,on,off,over,under,again,further,then,once,here,there,when,where,why,how,all,any,both,each,few,more,most,other,some,such,no,nor,not,only,own,same,so,than,too,very,say,says,said,shall";

    var word_count = {};

    if (words.length == 1) {
        word_count[words[0]] = 1;
    } else {
        words.forEach(function (word) {
            var word = word.toLowerCase();
            if (word != "" && common.indexOf(word) == -1 && word.length > 2) {
                if (word_count[word]) {
                    word_count[word]++;
                } else {
                    word_count[word] = 1;
                }
            }
        });
    }

    return d3.entries(word_count);
}

// Generate the word cloud based on the currently selected images' captions
function word_cloud() {
    $('#word-cloud').empty();

    // var fill = d3.scaleOrdinal(d3.schemeCategory10);

    var fill = d3.scaleOrdinal().range(["#262626","#66D1CD","#01807D","#C6C4C4","#7C9EB2"]);

    var width = 500;
    var height = 500;

    var word_entries = get_word_entries();

    if (word_entries.length == 0) {
        $('#word-cloud').append("<div class='h2 empty-word-cloud'>Select images to generate the word cloud</div>");
        return;
    }

    var xScale = d3.scaleLinear()
        .domain([0, d3.max(word_entries, function(d) {
            return d.value;
            })
        ])
        .range([10,100]).clamp(true);

    var layout = d3.layout.cloud()
        .size([width, height])
        .words(word_entries)
        .text(function(d) { return d.key; })
        .rotate(function() { return ~~(Math.random() * 2) * 0; })
        .font("IBM Plex Sans")
        .fontSize(function(d) { return xScale(+d.value); })
        .on("end", draw);

    layout.start();

    function on_word_mouseover(d) {
        d3.select(this).style("font-size", function(d) {
            return xScale(+d.value) + 5 + "px";
        });
    }

    function on_word_mouseout(d) {
        d3.select(this).style("font-size", function(d) {
            return xScale(+d.value) + "px";
        });
    }

    function on_word_click(d) {
        select_on(d.text);
    }

    function draw(words) {
        d3.select('#word-cloud').append("svg")
                .attr("width", width)
                .attr("height", height)
            .append("g")
                .attr("transform", "translate(" + [width >> 1, height >> 1] + ")")
            .selectAll("text")
                .data(words)
            .enter().append("text")
                .style("font-size", function(d) { return xScale(+d.value) + "px"; })
                .style("font-family", "IBM Plex Sans")
                .style("cursor", "pointer")
                .style("fill", function(d, i) { return fill(i); })
                .attr("text-anchor", "middle")
                .attr("transform", function(d) {
                    return "translate(" + [d.x, d.y] + ")rotate(" + d.rotate + ")";
                })
                .text(function(d) { return d.key; })
                .on("mouseover", on_word_mouseover)
                .on("mouseout", on_word_mouseout)
                .on("click", on_word_click);
    }
}

// Set the selected images to a subset of the current selection containing the given word in their caption
function select_on(word) {
    var key_list = [];
    $('#thumbnails .image_picker_selector .selected').children().filter("img").each(function () {
        capt_text = $(this).attr('alt');
        if (capt_text.indexOf(word) >= 0) {
            key_list.push($(this).attr('src'))
        }
    });
    set_selected_images(key_list);
}

// Initialize the Image Picker
function set_img_picker() {
    $("#thumbnails select").imagepicker({
        show_label: true,
        initialized: function() {
            $(".thumbnail").each(function() {
                var img_file = $(this).children('img').attr('src');
                $("<a href='#'/>")
                    .attr("class", "glyphicon glyphicon-resize-full more-info-icon")
                    .attr("data-featherlight", "/detail?image=" + img_file + " .image-detail")
                    .prependTo($(this));
            });
            $('a.more-info-icon').featherlight('ajax');
        },
        clicked: function() {
            word_cloud();
        }
    });
}

// Set the selected images to the given list
function set_selected_images(imgs) {
    $("#thumbnails select").val(imgs);
    $("#thumbnails select").data('picker').sync_picker_with_select();
    word_cloud();
}

// Select or Deselect all images
function select_all(bool) {
    if (bool) {
        set_selected_images(get_keys());
    } else {
        set_selected_images([]);
    }
}

$(function() {
    set_img_picker();
    select_all(true);

    // Image upload form submit functionality
    $('#img-upload').on('submit', function(data){
        // Stop form from submitting normally
        event.preventDefault();

        // Get form data
        var form = event.target;
        var data = new FormData(form);

        if ($("#file-input").val() != "") {
            $("#file-submit").text("Uploading...")

            // Perform file upload
            $.ajax({
                url: "/upload",
                method: "post",
                processData: false,
                contentType: false,
                data: data,
                processData: false,
                dataType: "json",
                success: function(data) {
                    add_thumbnails(data);
                    set_img_picker();
                },
                error: function() {
                    alert("Must submit a valid file (png, jpeg, jpg, or gif)");
                },
                complete: function() {
                    $("#file-submit").text("Submit");
                    $("#file-input").val("");
                }
            })
        }
    });

    // Stop propagation so images aren't 'selected' when the more info icon is clicked
    $('.more-info-icon').on('click', function(e) {
        e.stopPropagation();
    });
});
