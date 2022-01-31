import os
import tempfile

from assertpy import assert_that

from routeviews_google_upload import client


def test_remove_leading_backslash():
    # Arrange
    with tempfile.NamedTemporaryFile() as file:
        assert_that(file.name).starts_with(os.sep)

        # Act
        payload = client.generate_FileRequest(file.name, False)

        # Assert
        first_non_os_sep_char = file.name[1]
        assert_that(payload.filename).starts_with(first_non_os_sep_char)


def test_simple_filename():
    # Arrange
    with tempfile.NamedTemporaryFile() as file:

        # Act
        payload = client.generate_FileRequest(file.name, False)

        # Assert
        assert_that(payload.filename).is_equal_to(file.name.lstrip(os.sep))


def test_override_filename():
    # Arrange
    with tempfile.NamedTemporaryFile() as file:
        override_filename = 'test/this/thing-with-a-different-filename.bz2'

        # Act
        payload = client.generate_FileRequest(file.name, False, override_filename)

        # Assert
        assert_that(payload.filename).is_equal_to(override_filename)
